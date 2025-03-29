//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/typestep
//

package main

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsevents"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/fogfish/scud"
	"github.com/fogfish/typestep"
	"github.com/fogfish/typestep/examples/internal/core"
)

func main() {
	app := awscdk.NewApp(nil)
	stack := awscdk.NewStack(app, jsii.String("example-typestep"), nil)
	input := awsevents.NewEventBus(stack, jsii.String("Input"),
		&awsevents.EventBusProps{
			EventBusName: awscdk.Aws_STACK_NAME(),
		},
	)

	reply := awssqs.NewQueue(stack, jsii.String("Reply"),
		&awssqs.QueueProps{
			QueueName: awscdk.Aws_STACK_NAME(),
		},
	)

	//
	// Declare AWS Step Function using typestep constructs
	//
	a := typestep.From[core.Account](input)
	b := typestep.Join(core.GetUser, f(stack, "AtoU"), a)
	c := typestep.Join(core.PickCategory, f(stack, "UtoCs"), b)
	d := typestep.Lift(core.PickProduct, f(stack, "CtoPs"), c)
	e := typestep.Lift(core.MailTo, f(stack, "PtoS"), d)
	f := typestep.ToQueue(reply, e)

	ts := typestep.NewTypeStep(stack, jsii.String("Pipe"),
		&typestep.TypeStepProps{
			DeadLetterQueue: reply,
		},
	)
	typestep.StateMachine(ts, f)

	app.Synth(nil)
}

// Helper function to create AWS Lambda.
// Just created to reduce code duplication
func f(scope constructs.Construct, id string) awslambda.Function {
	return scud.NewFunctionGo(scope, jsii.String(id),
		&scud.FunctionGoProps{
			SourceCodeModule: "github.com/fogfish/typestep",
			SourceCodeLambda: "examples/cmd/f" + id,
		},
	)
}
