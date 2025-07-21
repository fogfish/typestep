//
// Copyright (C) 2025 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/typestep
//

package typestep

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2/awsevents"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseventstargets"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssqs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsstepfunctions"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsstepfunctionstasks"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/fogfish/golem/duct"
)

// F is a generic interface that represents a function from A to B
// and its associated AWS Lambda implementation.
//
// It's phantom method HKT1, which encodes the type-level information
// of a rank-1 function type func(A) B. This serves as a placeholder
// and ensures that implementations respect the intended type signature.
//
// An actual AWS Lambda implementation through the F() method.
type F[A, B any] interface {
	// HKT1 is a phantom method that represents the type-level
	// information of a function A → B. It is not meant to be called.
	HKT1(func(A) B)

	// F returns the underlying AWS Lambda IFunction instance.
	F() awslambda.IFunction
}

// Creates new morphism 𝑚, binding it with EventBridge for reading category `A` events.
func From[A any](in awsevents.IEventBus, cat ...string) duct.Morphism[A, A] {
	return duct.From(duct.L1[A](source{cat: cat, bus: in}))
}

type source struct {
	cat []string
	bus awsevents.IEventBus
}

// Compose lambda function transformer 𝑓: B ⟼ C with morphism 𝑚: A ⟼ B producing a new morphism 𝑚: A ⟼ C.
func Join[A, B, C any](
	f F[B, C],
	m duct.Morphism[A, B],
) duct.Morphism[A, C] {
	fn := lambda{concurency: 1, f: f.F()}
	return duct.Join(duct.L2[B, C](fn), m)
}

type lambda struct {
	concurency int
	f          awslambda.IFunction
}

// Compose lambda function transformer 𝑓: B ⟼ C with morphism 𝑚: A ⟼ []B.
// It produces a new computation 𝑚: A ⟼ []C that enables transformation within
// `[]C` context without immediate collapsing (see [duct.LiftF] for details).
// In other words, it nests the computation within the slice context.
// It is a responsibility of lifter to do something with those nested contexts
// either yielding individual elements or uniting (e.g. use Unit(Join(g, Lift(f)))
// to leave nested context into the morphism 𝑚: A ⟼ []C).
func Lift[A, B, C any](
	f F[B, C],
	m duct.Morphism[A, []B],
) duct.Morphism[A, C] {
	fn := lambda{concurency: 1, f: f.F()}
	return duct.LiftF(duct.L2[B, C](fn), m)
}

// See [Lift] for details. The function LiftP is equivalent to Lift but allows
// to specify the maximum number of concurrent invocations of the lambda function.
func LiftP[A, B, C any](
	n int,
	f F[B, C],
	m duct.Morphism[A, []B],
) duct.Morphism[A, C] {
	fn := lambda{concurency: n, f: f.F()}
	return duct.LiftF(duct.L2[B, C](fn), m)
}

// Wrap is equivalent to Lift but operates directly on the inner structure of
// the morphism 𝑚: A ⟼ []B, extracting individual elements of B while
// preserving the transformation context, enabling further composition.
//
// Usable to Yield elements of []B without transformation
func Wrap[A, B any](m duct.Morphism[A, []B]) duct.Morphism[A, B] {
	return duct.WrapF(m)
}

// Unit finalizes a transformation context by collapsing the nested morphism.
// It acts as the terminal operation, ensuring that all staged compositions,
// such as those built with Lift and Wrap, are fully resolved into a single,
// consumable form.
func Unit[A, B any](m duct.Morphism[A, B]) duct.Morphism[A, []B] {
	return duct.Unit(m)
}

// Yield results of 𝑚: A ⟼ B binding it with AWS SQS.
func ToQueue[A, B any](q awssqs.IQueue, m duct.Morphism[A, B]) duct.Morphism[A, duct.Void] {
	return duct.Yield(duct.L1[B](q), m)
}

// Yield results of 𝑚: A ⟼ B binding it with AWS EventBridge.
func ToEventBus[A, B any](source string, bus awsevents.IEventBus, m duct.Morphism[A, B], cat ...string) duct.Morphism[A, duct.Void] {
	return duct.Yield(duct.L1[B](eventbus{bus: bus, source: source, cat: cat}), m)
}

type eventbus struct {
	bus    awsevents.IEventBus
	source string
	cat    []string
}

//------------------------------------------------------------------------------

// TypeStep is AWS CDK L3, a builder for AWS Step Function state machine.
type TypeStep interface {
	constructs.IConstruct
}

// TypeStep L3 construct properties
type TypeStepProps struct {
	// DeadLetterQueue is the queue to receive messages to if an error occurs
	// while running the computation. The message is input JSON and "error".
	DeadLetterQueue awssqs.IQueue

	// SeqConcurrency is the maximum number of lambda's invocations allowed for
	// itterators while processing the sequence of computations (morphism 𝑚: A ⟼ []B).
	SeqConcurrency *float64
}

// private type - duct ast builder
type typeStep struct {
	constructs.Construct
	DeadLetterQueue awssqs.IQueue
	bus             awsevents.IEventBus
	eventPattern    *awsevents.EventPattern
	args            string
	stack           []awsstepfunctions.Chain
	names           []string
}

type node interface {
	constructs.IConstruct
	awsstepfunctions.INextable
	awsstepfunctions.IChainable
}

var _ duct.Visitor = (*typeStep)(nil)

// Create a new instance of TypeStep construct
func NewTypeStep(scope constructs.Construct, id *string, props *TypeStepProps) TypeStep {
	builder := &typeStep{
		Construct:       constructs.NewConstruct(scope, id),
		DeadLetterQueue: props.DeadLetterQueue,
		stack:           []awsstepfunctions.Chain{nil},
		names:           []string{""},
	}
	return builder
}

// StateMachine injects the morphism into the AWS Step Function,
// it constructs the state machine from the defined computation.
func StateMachine[A, B any](ts TypeStep, m duct.Morphism[A, B]) {
	b := ts.(*typeStep)
	if err := m.Apply(b); err != nil {
		panic(err)
	}
}

func (ts *typeStep) append(f node) {
	tsal := len(ts.stack) - 1
	last := ts.stack[tsal]
	if last == nil {
		ts.stack[tsal] = awsstepfunctions.Chain_Start(f)
	} else {
		ts.stack[tsal] = last.Next(f)
	}
	ts.names[tsal] = ts.names[tsal] + *f.Node().Id()
}

func (ts *typeStep) OnEnterMorphism(depth int, node duct.AstSeq) error {
	return nil
}

func (ts *typeStep) OnLeaveMorphism(depth int, node duct.AstSeq) error {
	if len(ts.stack) != 1 {
		return fmt.Errorf("bad definition of compute pipeline")
	}

	if ts.bus == nil {
		return fmt.Errorf("undefined event source for compute pipeline")
	}

	states := awsstepfunctions.NewStateMachine(ts.Construct, jsii.String("StateMachine"),
		&awsstepfunctions.StateMachineProps{
			DefinitionBody: awsstepfunctions.ChainDefinitionBody_FromChainable(ts.stack[0]),
		},
	)

	awsevents.NewRule(ts.Construct, jsii.String("Rule"),
		&awsevents.RuleProps{
			EventBus:     ts.bus,
			EventPattern: ts.eventPattern,
		},
	).AddTarget(
		awseventstargets.NewSfnStateMachine(
			states,
			&awseventstargets.SfnStateMachineProps{},
		),
	)

	return nil
}

func (ts *typeStep) OnEnterSeq(depth int, node duct.AstSeq) error {
	ts.stack = append(ts.stack, nil)
	ts.names = append(ts.names, "")
	ts.args = "$"

	return nil
}

func (ts *typeStep) OnLeaveSeq(depth int, node duct.AstSeq) error {
	last := len(ts.stack) - 1

	name := ts.names[last]
	hash := sha256.Sum256([]byte(name))
	ihex := hex.EncodeToString(hash[:])[:8]

	concurency := 1
	if f, ok := node.Seq[0].(duct.AstMap); ok {
		if f, ok := f.F.(lambda); ok {
			concurency = f.concurency
		}
	}

	foreach := awsstepfunctions.NewMap(ts.Construct, jsii.String("Seq"+ihex),
		&awsstepfunctions.MapProps{
			ItemsPath:      jsii.String("$.Payload"), // assuming the first element is function, which is true by defsign
			MaxConcurrency: jsii.Number(concurency),
		},
	)

	foreach.ItemProcessor(ts.stack[last],
		&awsstepfunctions.ProcessorConfig{},
	)

	ts.stack = ts.stack[:last]
	ts.names = ts.names[:last]
	ts.append(foreach)
	ts.args = "$"

	return nil
}

func (ts *typeStep) OnEnterMap(depth int, node duct.AstMap) error {
	switch f := node.F.(type) {
	case lambda:
		uuid := *f.f.Node().Id()
		compute := awsstepfunctionstasks.NewLambdaInvoke(
			ts.Construct,
			jsii.String("Map"+uuid),
			&awsstepfunctionstasks.LambdaInvokeProps{
				InputPath:      jsii.String(ts.args),
				LambdaFunction: f.f,
			},
		)

		if ts.DeadLetterQueue != nil {
			dlq := awsstepfunctionstasks.NewSqsSendMessage(ts.Construct, jsii.String("Try"+uuid),
				&awsstepfunctionstasks.SqsSendMessageProps{
					Queue:       ts.DeadLetterQueue,
					MessageBody: awsstepfunctions.TaskInput_FromJsonPathAt(jsii.String("$")),
				},
			)
			err := awsstepfunctions.NewFail(ts.Construct, jsii.String("Err"+uuid),
				&awsstepfunctions.FailProps{},
			)

			compute.AddCatch(
				dlq.Next(err),
				&awsstepfunctions.CatchProps{
					ResultPath: jsii.String("$.error"),
				},
			)
		}

		ts.append(compute)
		return nil
	default:
		return fmt.Errorf("unkown compute type: %T", f)
	}
}

func (ts *typeStep) OnLeaveMap(depth int, node duct.AstMap) error {
	// Note: Lambda's response of step function is always packed
	ts.args = "$.Payload"
	return nil
}

func (ts *typeStep) OnEnterFrom(depth int, node duct.AstFrom) error {
	switch f := node.Source.(type) {
	case source:
		ts.bus = f.bus
		ts.eventPattern = &awsevents.EventPattern{
			DetailType: jsii.Strings(node.Type),
		}
		if len(f.cat) != 0 {
			ts.eventPattern.DetailType = jsii.Strings(f.cat...)
		}
		ts.args = "$.detail"
		return nil
	default:
		return fmt.Errorf("unkown input type: %T", f)
	}
}

func (ts *typeStep) OnLeaveFrom(depth int, node duct.AstFrom) error {
	return nil
}

func (ts *typeStep) OnEnterYield(depth int, node duct.AstYield) error {
	switch f := node.Target.(type) {
	case awssqs.IQueue:
		sink := awsstepfunctionstasks.NewSqsSendMessage(ts.Construct, jsii.String("Sink"),
			&awsstepfunctionstasks.SqsSendMessageProps{
				Queue:       f,
				MessageBody: awsstepfunctions.TaskInput_FromJsonPathAt(jsii.String(ts.args)),
			},
		)
		ts.append(sink)
		return nil

	case eventbus:
		kind := node.Type
		if len(f.cat) != 0 {
			kind = f.cat[0]
		}

		sink := awsstepfunctionstasks.NewEventBridgePutEvents(ts.Construct, jsii.String("Sink"),
			&awsstepfunctionstasks.EventBridgePutEventsProps{
				Entries: &[]*awsstepfunctionstasks.EventBridgePutEventsEntry{
					{
						Detail:     awsstepfunctions.TaskInput_FromJsonPathAt(jsii.String(ts.args)),
						DetailType: jsii.String(kind),
						Source:     jsii.String(f.source),
						EventBus:   f.bus,
					},
				},
			},
		)
		ts.append(sink)
		return nil

	default:
		return fmt.Errorf("unkown reply type: %T", f)
	}
}

func (ts *typeStep) OnLeaveYield(depth int, node duct.AstYield) error {
	return nil
}
