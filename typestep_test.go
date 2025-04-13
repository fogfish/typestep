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
	"github.com/aws/aws-cdk-go/awscdk/v2/awsevents"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/jsii-runtime-go"
	"github.com/fogfish/typestep"
)

func TestTypeStep(t *testing.T) {
	// GIVEN
	app := awscdk.NewApp(nil)
	stack := awscdk.NewStack(app, jsii.String("Test"), nil)
	event := awsevents.EventBus_FromEventBusArn(stack, jsii.String("Events"), jsii.String("arn:aws:events:eu-west-1:000000000000:event-bus:my-event-bus"))
	queue := awssqs.Queue_FromQueueArn(stack, jsii.String("Queue"), jsii.String("arn:aws:sqs:eu-west-1:000000000000:my-queue"))

	a := typestep.Function_FromFunctionArn[string, string](stack, jsii.String("A"),
		jsii.String("arn:aws:lambda:eu-west-1:000000000000:function:my-function"))

	b := typestep.Function_FromFunctionArn[string, []string](stack, jsii.String("B"),
		jsii.String("arn:aws:lambda:eu-west-1:000000000000:function:my-function"))

	c := typestep.Function_FromFunctionArn[string, []string](stack, jsii.String("C"),
		jsii.String("arn:aws:lambda:eu-west-1:000000000000:function:my-function"))

	d := typestep.Function_FromFunctionArn[string, string](stack, jsii.String("D"),
		jsii.String("arn:aws:lambda:eu-west-1:000000000000:function:my-function"))

	// THEN
	p1 := typestep.From[string](event)
	p2 := typestep.Join(a, p1)
	p3 := typestep.Join(b, p2)
	p4 := typestep.Lift(c, p3)
	p5 := typestep.Lift(d, p4)
	p6 := typestep.ToQueue(queue, p5)

	ts := typestep.NewTypeStep(stack, jsii.String("Pipe"),
		&typestep.TypeStepProps{
			DeadLetterQueue: queue,
		},
	)
	typestep.StateMachine(ts, p6)

	// WHEN
	require := map[*string]*float64{
		jsii.String("AWS::Events::Rule"):                jsii.Number(1),
		jsii.String("AWS::StepFunctions::StateMachine"): jsii.Number(1),
		jsii.String("AWS::IAM::Role"):                   jsii.Number(2),
	}

	template := assertions.Template_FromStack(stack, nil)
	for key, val := range require {
		template.ResourceCountIs(key, val)
	}
}
