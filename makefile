setup:
	brew install ngrok/ngrok/ngrok

start:
	go run main.go

ngrok:
	ngrok http 8080