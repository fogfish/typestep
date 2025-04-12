<p align="center">
  <h3 align="center">⟦𝚲 𝜏 . 𝜏⟧</h3>
  <h3 align="center">typestep</h3>
  <p align="center"><strong>type-safe choreography for AWS Step Functions</strong></p>

  <p align="center">
    <!-- Version -->
    <a href="https://github.com/fogfish/typestep/releases">
      <img src="https://img.shields.io/github/v/tag/fogfish/typestep?label=version" />
    </a>
    <!-- Documentation -->
    <a href="https://pkg.go.dev/github.com/fogfish/typestep">
      <img src="https://pkg.go.dev/badge/github.com/fogfish/typestep" />
    </a>
    <!-- Build Status  -->
    <a href="https://github.com/fogfish/typestep/actions/">
      <img src="https://github.com/fogfish/typestep/workflows/build/badge.svg" />
    </a>
    <!-- GitHub -->
    <a href="http://github.com/fogfish/typestep">
      <img src="https://img.shields.io/github/last-commit/fogfish/typestep.svg" />
    </a>
    <!-- Coverage -->
    <a href="https://coveralls.io/github/fogfish/typestep?branch=main">
      <img src="https://coveralls.io/repos/github/fogfish/typestep/badge.svg?branch=main" />
    </a>
    <!-- Go Card -->
    <a href="https://goreportcard.com/report/github.com/fogfish/typestep">
      <img src="https://goreportcard.com/badge/github.com/fogfish/typestep" />
    </a>
  </p>
</p>

--- 

This library provides AWS CDK L3 constructs for defining AWS Step Functions using a type-safe notation in Go.  

## Inspiration

[AWS Step Functions](https://docs.aws.amazon.com/step-functions/latest/dg/welcome.html) provide an out-of-the-box implementation of the [choreography pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/choreography). Their seamless integration with AWS services, especially AWS Lambda, makes them the default choice for AWS-hosted workloads. The main challenge, however, lies in the complexity of workflow specification and it further maintainability. Amazon has developed a domain specific language for definition of the state machine structure—ultimately requiring you to 'code' and testing in JSON. While AWS CDK improves this experience with its L2 constructs, allowing workflow choreography to be defined using general-purpose languages like TypeScript, Go, and others, the process can still be complex.

The biggest challenge is the reliance on duck typing when composing Lambdas into the workflow. A single refactoring mistake in one function can break the entire workflow, with issues only becoming visible at runtime—there is no compile-time inference when composing Lambda A with Lambda B.
 
**`typestep`** is a lightweight library designed to simplify the definition of state machines for AWS Step Functions. By introducing a **type-safe notation**, it eliminates the challenges of duck typing and ensures **compile-time inference** of AWS Lambda signatures. This approach enhances reliability, making workflow choreography **easier to define, maintain, and refactor** while reducing runtime errors.

## Getting Starter

- [Inspiration](#inspiration)
- [Getting Starter](#getting-starter)
  - [Quick example](#quick-example)
  - [Type-safe annotation of AWS Lambda](#type-safe-annotation-of-aws-lambda)
  - [Workflow composition](#workflow-composition)
    - [*Form* sources events](#form-sources-events)
    - [*Join* composes functions](#join-composes-functions)
    - [*Lift*, *Wrap* and *Unit* builds nested computations](#lift-wrap-and-unit-builds-nested-computations)
    - [*Yield* the results](#yield-the-results)
- [How To Contribute](#how-to-contribute)
- [License](#license)


The latest version of the library is available at `main` branch of this repository. All development, including new features and bug fixes, take place on the `main` branch using forking and pull requests as described in contribution guidelines. The stable version is available via Golang modules.

Use go get to retrieve the library and add it as dependency to your application.

```bash
go get -u github.com/fogfish/typestep
```

### Quick example

Example below is most simplest illustration on how to make a type-safe composition of lambda function into AWS Step Function workflow.

```go
a := typestep.From[string](
  awsevents.EventBus_FromEventBusArn(/* ... */),
)

b := typestep.Join(
  func(name string) (string, error) { /* ... */ },
  awslambda.Function_FromFunctionArn(/* ... */),
  a,
)

c := typestep.ToQueue(
  awssqs.Queue_FromQueueArn(/* ... */),
  b,
)

workflow := typestep.NewTypeStep(stack, jsii.String("Workflow"),
  &typestep.TypeStepProps{},
)
typestep.StateMachine(workflow, c)
```

More detailed examples are [here](./examples/)

### Type-safe annotation of AWS Lambda

AWS Lambda does not impose restrictions on the development runtime, allowing the use of type-safe languages like Go. However, outside of the function itself, type safety is not enforced by the AWS environment, as Lambda relies on JSON for input and output handling. As a consequence the duck typing is used when composing Lambdas. When building Go-based workflows, Lambdas must be lifted into a type-safe abstraction. Since Lambda is merely a deployment pattern, it is recommended to define workflow functions within the core domain and reference them in both the Lambda configuration and infrastructure-as-code (IaC) definitions.  

Unlike a typical AWS Lambda deployment where `func main()` serves as the entry point, this library demands a function that returns a valid AWS Lambda handler of the form:

```go
func Main() func(context.Context, A) (B, error) {
  /* AWS Lambda bootstrap code goes here */
  return func(context.Context, A) (B, error) {
    /* AWS Lambda handler goes here */
  }
}
```

The primary reason is that the library automatically generates a `main.go` file from the provided handler, ensuring consistent wiring and preserving type information throughout the deployment and execution.

```go
// app/internal/core/biz.go
func GetUser(ctx context.Context, acc Account) (User, error) { /* ... */ } 

// app/cmd/lambda/main.go
func Factory() func(ctx context.Context, acc Account) (User, error) { return GetUser } 

// app/internal/cdk/workflow.go

// declares AWS Lambda resource
f := typestep.NewFunctionTyped(stack, jsii.String("Lambda"),
  typestep.FunctionTyped(Factory,
    &scud.FunctionGoProps{
      SourceCodeModule: "github.com/fogfish/app",
      SourceCodeLambda: "cmd/lambda",
    },
  ),
)

// use AWS Lambda with type-safe signature inside the workflow
typestep.Join(f, /* ... */)
```

This technique allows validation of function signatures at compile time.

### Workflow composition

The library uses category-theory-inspired algebra defined [here](https://github.com/fogfish/golem/tree/main/duct) to compose workflows. Its algebra is tailored for effective composition of `ƒ: A ⟼ B` and `ƒ: A ⟼ []B` types of computations.

The workflow is triggered by the AWS EventBridge event and passes through a series of transformations defined by AWS Lambda functions. The results are then either emitted back to AWS EventBridge or sent to an AWS SQS queue. Any errors encountered during execution are captured in a dead-letter queue (AWS SQS) for further analysis. The library does not provide L3 constructs for provisioning AWS EventBridge, Lambda, or SQS. Its sole focus is on defining AWS Step Functions and their state machines.

The library provide simple api for the workflow composition: `From`, `Join`, `Lift`, `Wrap`, `Unit` and `Yeild`.

Once the workflow is composed, deploy it using `TypeStep` L3 construct:

```go
// Declare the workflow
a := typestep.From[core.Account](input)
// ...
f := typestep.ToQueue(reply, e)

// Deploy the workflow
ts := typestep.NewTypeStep(stack, jsii.String("Pipe"), &typestep.TypeStepProps{})
typestep.StateMachine(ts, f)
```

#### *Form* sources events

`From` binds EventBridge to an AWS Step Function, automatically configuring the consumption of all events where `detail-type` matches the specified type name. For example, in the snippet below, all events with `detail-type` set to `Account` will trigger the computation.

```go
bus := awsevents.NewEventBus(/* ... */)
// ...
a := typestep.From[core.Account](bus)
```

#### *Join* composes functions

The simple operation above returns a workflow definition that represents an identity function `ƒ: Account ⟼ Account`. It can be further composed with any function of type `𝑔: Account ⟼ ?`, using `Join`.

```go
func GetUser(Account) (User, error) { /* ... */ }

fun := awslambda.NewFunction(/* ... */)

b := typestep.Join(GetUser, fun, a)
```

#### *Lift*, *Wrap* and *Unit* builds nested computations

If your first function returns a list (`ƒ: A ⟼ []B`) and needs to be composed with `𝑔: B ⟼ C`, you must lift the computation to ensure proper composition.

```go
func GetManyB(A) ([]B, error) { /* ... */ }
func UseJustB(B) (C, error) { /* ... */ }

b := typestep.Join(GetManyB, /* ... */)
c := typestep.Lift(UseJustB, /* ... */, b)
// ... 
x := typestep.Unit(/* ... */)
```

In the functional programming, this abstraction is called "free monad". We are lifting the function `UseJustB` within "a functorial context"--`[]B` list. It is a responsibility of creator of such op to do something with those nested stucts either yielding individual elements (`ToQueue`, `ToEventBus`) or uniting it. Think about it as the following construct.  

```go
for _, b := range GetManyB() {
  UseJustB(b)
}
```

#### *Yield* the results

The workflow completes by emitting an event to AWS SQS or EventBridge, unless explicitly persisted elsewhere through a chained AWS Lambda function. 

```go
x := typestep.ToQueue(/* ... */)
```

## How To Contribute

The library is [MIT](LICENSE) licensed and accepts contributions via GitHub pull requests:

1. Fork it
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Added some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create new Pull Request


The build and testing process requires [Go](https://golang.org) version 1.24 or later.

**Build** and **run** in your development console.

```bash
git clone https://github.com/fogfish/typestep
go test ./...
```

## License

[![See LICENSE](https://img.shields.io/github/license/fogfish/typestep.svg?style=for-the-badge)](LICENSE)
