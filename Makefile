default: invoke

EVENT=event.json

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ./bootstrap main.go

zip:
	zip -j function.zip ./bootstrap

sam-build: build zip
	sam build

invoke: sam-build
	sam local invoke \
	--invoke-image amazon/aws-lambda-provided:al2 \
	--event events/${EVENT}
	
start-api: sam-build
	sam local start-api