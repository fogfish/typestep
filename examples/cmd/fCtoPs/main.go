package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/fogfish/typestep/examples/internal/core"
)

func main() {
	lambda.Start(core.PickProduct)
}
