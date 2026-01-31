package internal

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type StackRef struct {
	Stack *pulumi.StackReference
}

type StackRefFuncs interface {
	Out() pulumi.AnyOutput
	Details() *pulumi.StackReferenceOutputDetails
	Str() pulumi.StringOutput
	Int() pulumi.IntOutput
	Init() *pulumi.StackReference
}

func (st *StackRef) Init(ctx *pulumi.Context, name, slug string) {
	stk, err := pulumi.NewStackReference(ctx, name, &pulumi.StackReferenceArgs{
		Name: pulumi.String(slug),
	})
	if err != nil {
		msg := "failed to create new stack reference: " + err.Error()
		_ = ctx.Log.Error(msg, nil)
	}

	st.Stack = stk
}

func (st *StackRef) Out(v string) pulumi.AnyOutput {
	return st.Stack.GetOutput(pulumi.String(v))
}

func (st *StackRef) Details(v string) *pulumi.StackReferenceOutputDetails {
	out, err := st.Stack.GetOutputDetails(v)
	if err != nil {
		return nil
	}

	return out
}

func (st *StackRef) Id(v string) pulumi.IDOutput {
	return st.Stack.GetIDOutput(pulumi.String(v))
}

func (st *StackRef) Int(v string) pulumi.IntOutput {
	return st.Stack.GetIntOutput(pulumi.String(v))
}

func (st *StackRef) Str(v string) pulumi.StringOutput {
	return st.Stack.GetStringOutput(pulumi.String(v))
}
