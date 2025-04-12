package test

import (
	"context"

	_ "github.com/aws/aws-lambda-go/lambda"
)

func Main() func(context.Context, string) (string, error) {
	return func(ctx context.Context, s string) (string, error) {
		return "", nil
	}
}
