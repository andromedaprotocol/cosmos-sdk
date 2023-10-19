package implementation

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
)

// Dependencies are passed to the constructor of a smart account.
type Dependencies struct {
	SchemaBuilder *collections.SchemaBuilder
	AddressCodec  address.Codec
}

// AccountCreatorFunc is a function that creates an account.
type AccountCreatorFunc = func(deps Dependencies) (string, Implementation, error)

// AddAccount is a helper function to add a smart account to the list of smart accounts.
func AddAccount[A Account](name string, constructor func(deps Dependencies) (A, error)) func(deps Dependencies) (string, Implementation, error) {
	return func(deps Dependencies) (string, Implementation, error) {
		acc, err := constructor(deps)
		if err != nil {
			return "", Implementation{}, err
		}
		impl, err := NewImplementation(acc)
		if err != nil {
			return "", Implementation{}, err
		}
		return name, impl, nil
	}
}

func MakeAccountsMap(addressCodec address.Codec, accounts []AccountCreatorFunc) (map[string]Implementation, error) {
	accountsMap := make(map[string]Implementation, len(accounts))
	for _, makeAccount := range accounts {
		stateSchemaBuilder := collections.NewSchemaBuilderFromAccessor(OpenKVStore)
		deps := Dependencies{
			SchemaBuilder: stateSchemaBuilder,
			AddressCodec:  addressCodec,
		}
		name, impl, err := makeAccount(deps)
		if err != nil {
			return nil, fmt.Errorf("failed to create account %s: %w", name, err)
		}
		// build schema
		schema, err := stateSchemaBuilder.Build()
		if err != nil {
			return nil, fmt.Errorf("failed to build schema for account %s: %w", name, err)
		}
		impl.CollectionsSchema = schema
		// check if account name is already registered
		if _, ok := accountsMap[name]; ok {
			return nil, fmt.Errorf("account %s is already registered", name)
		}
		accountsMap[name] = impl
	}

	return accountsMap, nil
}

// NewImplementation creates a new Implementation instance given an Account implementer.
func NewImplementation(account Account) (Implementation, error) {
	// make init handler
	ir := NewInitBuilder()
	account.RegisterInitHandler(ir)
	initHandler, err := ir.makeHandler()
	if err != nil {
		return Implementation{}, err
	}

	// make execute handler
	er := NewExecuteBuilder()
	account.RegisterExecuteHandlers(er)
	executeHandler, err := er.makeHandler()
	if err != nil {
		return Implementation{}, err
	}

	// make query handler
	qr := NewQueryBuilder()
	account.RegisterQueryHandlers(qr)
	queryHandler, err := qr.makeHandler()
	if err != nil {
		return Implementation{}, err
	}
	return Implementation{
		Init:                  initHandler,
		Execute:               executeHandler,
		Query:                 queryHandler,
		InitRequestSchema:     ir.RequestSchema,
		InitResponseSchema:    ir.ResponseSchema,
		DecodeExecuteRequest:  er.makeRequestDecoder(),
		EncodeExecuteResponse: er.makeResponseEncoder(),
		DecodeQueryRequest:    qr.er.makeRequestDecoder(),
		EncodeQueryResponse:   qr.er.makeResponseEncoder(),
	}, nil
}

// Implementation wraps an Account implementer in order to provide a concrete
// and non-generic implementation usable by the x/accounts module.
type Implementation struct {
	// Init defines the initialisation handler for the smart account.
	Init func(ctx context.Context, msg any) (resp any, err error)
	// Execute defines the execution handler for the smart account.
	Execute func(ctx context.Context, msg any) (resp any, err error)
	// Query defines the query handler for the smart account.
	Query func(ctx context.Context, msg any) (resp any, err error)

	// Schema

	CollectionsSchema collections.Schema

	// InitRequestSchema is the MessageSchema of the init request.
	InitRequestSchema MessageSchema
	// InitResponseSchema is the MessageSchema of the init response.
	InitResponseSchema MessageSchema

	// DecodeExecuteRequest decodes an execute request coming from the message server.
	DecodeExecuteRequest func([]byte) (any, error)
	// EncodeExecuteResponse encodes an execute response to be sent back from the message server.
	EncodeExecuteResponse func(any) ([]byte, error)

	// DecodeQueryRequest decodes a query request coming from the message server.
	DecodeQueryRequest func([]byte) (any, error)
	// EncodeQueryResponse encodes a query response to be sent back from the message server.
	EncodeQueryResponse func(any) ([]byte, error)
}

// MessageSchema defines the schema of a message.
// A message can also define a state schema.
type MessageSchema struct {
	// Name identifies the message name, this must be queriable from some reflection service.
	Name string
	// TxDecode decodes into the message from transaction bytes.
	// CONSENSUS SAFE.
	TxDecode func([]byte) (any, error)
	// TxEncode encodes the message into transaction bytes.
	// CONSENSUS SAFE.
	TxEncode func(any) ([]byte, error)
	// HumanDecode decodes into the message from human-readable bytes.
	// CONSENSUS UNSAFE.
	HumanDecode func([]byte) (any, error)
	// HumanEncode encodes the message into human-readable bytes.
	// CONSENSUS UNSAFE.
	HumanEncode func(any) ([]byte, error)
}
