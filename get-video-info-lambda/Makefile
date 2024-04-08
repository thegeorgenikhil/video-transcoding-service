zipFile = myFunction.zip
executable = bootstrap
functionName = getVideoInfoLambda

ROLE = ${GET_VIDEO_INFO_LAMBDA_ROLE}

COLOUR_GREEN=\033[0;32m
COLOUR_RED=\033[0;31m
COLOUR_BLUE=\033[0;34m
END_COLOUR=\033[0m

hello:
	@echo "Hello, World"

build:
	@echo "Building the go binary"
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o $(executable) -tags lambda.norpc main.go

zip:
	@echo "Zipping the binary"
	zip $(zipFile) $(executable)

create_lambda: build zip
	@if [ -z $(ROLE) ]; then \
		echo "$(COLOUR_RED)ERROR: Please set the GET_VIDEO_INFO_LAMBDA_ROLE environment variable$(END_COLOUR)"; \
		echo "$(COLOUR_RED) To set the role, run the following command:$(END_COLOUR)"; \
		echo "$(COLOUR_RED) 	export GET_VIDEO_INFO_LAMBDA_ROLE=<role-arn>$(END_COLOUR)"; \
		exit 1; \
	fi

	@echo "Creating the lambda function"
	aws lambda create-function --function-name $(functionName) \
	--runtime provided.al2023 --handler $(executable) \
	--architectures arm64 \
	--role $(ROLE) \
	--zip-file fileb://$(zipFile)

update_lambda: build zip
	@echo "Updating the lambda function"
	aws lambda update-function-code --function-name $(functionName) \
	 --zip-file fileb://$(zipFile)
