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
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
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
	// Declare AWS Lambda with type safe annotations
	a2u := typestep.NewFunctionTyped(stack, jsii.String("AtoU"),
		typestep.NewFunctionTypedProps(core.GetUserF, &scud.FunctionGoProps{
			SourceCodeModule: "github.com/fogfish/typestep",
			SourceCodeLambda: "examples/cmd/fAtoU",
		}),
	)

	u2cs := typestep.NewFunctionTyped(stack, jsii.String("UtoCs"),
		typestep.NewFunctionTypedProps(core.PickCategoryF, &scud.FunctionGoProps{
			SourceCodeModule: "github.com/fogfish/typestep",
			SourceCodeLambda: "examples/cmd/fUtoCs",
		}),
	)

	c2ps := typestep.NewFunctionTyped(stack, jsii.String("CtoPs"),
		typestep.NewFunctionTypedProps(core.PickProductF, &scud.FunctionGoProps{
			SourceCodeModule: "github.com/fogfish/typestep",
			SourceCodeLambda: "examples/cmd/fCtoPs",
		}),
	)

	p2s := typestep.NewFunctionTyped(stack, jsii.String("PtoS"),
		typestep.NewFunctionTypedProps(core.MailToF, &scud.FunctionGoProps{
			SourceCodeModule: "github.com/fogfish/typestep",
			SourceCodeLambda: "examples/cmd/fPtoS",
		}),
	)

	//
	// Declare AWS Step Function using typestep constructs
	//
	a := typestep.From[core.Account](input)
	b := typestep.Join(a2u, a)
	c := typestep.Join(u2cs, b)
	d := typestep.Lift(c2ps, c)
	e := typestep.Lift(p2s, d)
	f := typestep.ToQueue(reply, e)

	ts := typestep.NewTypeStep(stack, jsii.String("Pipe"),
		&typestep.TypeStepProps{
			DeadLetterQueue: reply,
		},
	)
	typestep.StateMachine(ts, f)

	app.Synth(nil)
}
