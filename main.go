package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/elimity-com/scim"
	"github.com/elimity-com/scim/optional"
	scimSchema "github.com/elimity-com/scim/schema"
	"github.com/sirupsen/logrus"
	"github.com/wilkermichael/scim-prototype/handler"
)

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.Formatter = &logrus.TextFormatter{
		FullTimestamp: true,
	}
	logger.Info("Starting SCIM server")

	// Create a service provider configuration
	config := scim.ServiceProviderConfig{}

	// Create user schema
	s := scimSchema.Schema{
		ID:          "urn:ietf:params:scim:schemas:core:2.0:User",
		Name:        optional.NewString("User"),
		Description: optional.NewString("User Account"),
		Attributes: []scimSchema.CoreAttribute{
			scimSchema.SimpleCoreAttribute(scimSchema.SimpleStringParams(scimSchema.StringParams{
				Name:       "userName",
				Required:   true,
				Uniqueness: scimSchema.AttributeUniquenessServer(),
			})),
			scimSchema.SimpleCoreAttribute(scimSchema.SimpleStringParams(scimSchema.StringParams{
				Description: optional.NewString("A String that is an identifier for the resource as defined by the provisioning client."),
				Name:        "externalId",
				Uniqueness:  scimSchema.AttributeUniquenessServer(),
			})),
			scimSchema.SimpleCoreAttribute(scimSchema.SimpleStringParams(scimSchema.StringParams{
				Name: "nickName",
			})),
			scimSchema.SimpleCoreAttribute(scimSchema.SimpleBooleanParams(scimSchema.BooleanParams{
				Description: optional.NewString("A boolean denoting that the user is either active or disabled."),
				Name:        "active",
				Required:    false,
			})),
		},
	}

	extension := scimSchema.Schema{
		ID:          "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User",
		Name:        optional.NewString("EnterpriseUser"),
		Description: optional.NewString("Enterprise User"),
		Attributes: []scimSchema.CoreAttribute{
			scimSchema.SimpleCoreAttribute(scimSchema.SimpleStringParams(scimSchema.StringParams{
				Name: "employeeNumber",
			})),
			scimSchema.SimpleCoreAttribute(scimSchema.SimpleStringParams(scimSchema.StringParams{
				Name: "organization",
			})),
		},
	}

	resourceHandler := handler.NewUserResourceHandler(logger)

	// Create Resource Types
	resourceTypes := []scim.ResourceType{
		{
			ID:          optional.NewString("User"),
			Name:        "User",
			Endpoint:    "/Users",
			Description: optional.NewString("User Account"),
			Schema:      s,
			SchemaExtensions: []scim.SchemaExtension{
				{Schema: extension},
			},
			Handler: resourceHandler,
		},
	}

	// Create a new SCIM server
	serverArgs := scim.ServerArgs{
		ServiceProviderConfig: &config,
		ResourceTypes:         resourceTypes,
	}

	// Initialize a logger using logrus
	serverOpts := []scim.ServerOption{
		scim.WithLogger(logger),
	}

	server, err := scim.NewServer(&serverArgs, serverOpts...)
	if err != nil {
		logger.Fatalf("Failed to start SCIM server: %v", err)
	}

	// Register the SCIM server's HTTP handler at a specific path prefix.
	m := middleware{logger: logger}
	http.Handle("/scim/v2/", m.loggingMiddleware(http.StripPrefix("/scim/v2", server)))

	// Start the server
	logger.Info("SCIM server is running on http://localhost:8080/scim/v2/")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Fatalf("Failed to start SCIM server: %v", err)
	}
}

type middleware struct {
	logger *logrus.Logger
}

func (m middleware) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log the request
		m.logger.Printf("Received request: %s %s", r.Method, r.URL.Path)

		// Read the body
		if m.logger.Level == logrus.DebugLevel {
			switch r.Method {
			case http.MethodPost, http.MethodPatch, http.MethodPut:
				b, err := io.ReadAll(r.Body)
				if err != nil {
					m.logger.Errorf("Failed to read request body: %v", err)
				}

				var prettyJSON bytes.Buffer
				err = json.Indent(&prettyJSON, b, "", " \t")
				if err != nil {
					m.logger.Errorf("Failed to indent request body: %v", err)
				}
				m.logger.Debugf("Request body: \n%s", prettyJSON.String())

				// Replace read bytes
				r.Body = io.NopCloser(bytes.NewBuffer(b))
			}
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}
