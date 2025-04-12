//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/typestep
//

package typestep_test

import (
	"testing"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/assertions"
	"github.com/aws/jsii-runtime-go"
	"github.com/fogfish/scud"
	"github.com/fogfish/typestep"
	"github.com/fogfish/typestep/internal/test"
)

func TestFunctionTyped(t *testing.T) {
	// GIVEN
	app := awscdk.NewApp(nil)
	stack := awscdk.NewStack(app, jsii.String("Test"), nil)

	// THEN
	typestep.NewFunctionTyped(stack, jsii.String("T"),
		typestep.NewFunctionTypedProps(test.Main,
			&scud.FunctionGoProps{
				SourceCodeModule: "github.com/fogfish/typestep",
				SourceCodeLambda: "internal/test",
			},
		),
	)

	// WHEN
	require := map[*string]*float64{
		jsii.String("AWS::Lambda::Function"): jsii.Number(2),
	}

	template := assertions.Template_FromStack(stack, nil)
	for key, val := range require {
		template.ResourceCountIs(key, val)
	}

}
