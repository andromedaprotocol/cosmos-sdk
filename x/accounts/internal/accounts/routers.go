package accounts

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/emptypb"
)

type QueryRouter struct{ er *ExecuteRouter }

func (e *QueryRouter) Handler() (func(ctx context.Context, msg proto.Message) (proto.Message, error), error) {
	return e.er.Handler()
}

type ExecuteRouter struct {
	methodsMap map[protoreflect.FullName]func(ctx context.Context, msg proto.Message) (proto.Message, error)
	err        error
}

func (e *ExecuteRouter) Handler() (func(ctx context.Context, msg proto.Message) (proto.Message, error), error) {
	if e.err != nil {
		return nil, e.err
	}
	if e.methodsMap == nil {
		return func(_ context.Context, _ proto.Message) (proto.Message, error) {
			return nil, fmt.Errorf("this account does not accept execute messages")
		}, nil
	}
	return func(ctx context.Context, msg proto.Message) (proto.Message, error) {
		name := proto.MessageName(msg)
		handler, exists := e.methodsMap[name]
		if !exists {
			return nil, fmt.Errorf("unknown method %s", name)
		}
		resp, err := handler(ctx, msg)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}, nil
}

func RegisterQueryHandler[
	Req any, Resp any, ReqP Msg[Req], RespP Msg[Resp],
](router *QueryRouter, handler func(ctx context.Context, msg *Req) (*Resp, error)) {
	if router.er == nil {
		router.er = &ExecuteRouter{}
	}
	RegisterExecuteHandler[Req, Resp, ReqP, RespP](router.er, handler)
}

func RegisterExecuteHandler[
	Req any, Resp any, ReqP Msg[Req], RespP Msg[Resp],
](router *ExecuteRouter, handler func(ctx context.Context, msg *Req) (*Resp, error)) {
	if router.methodsMap == nil {
		router.methodsMap = make(map[protoreflect.FullName]func(ctx context.Context, msg proto.Message) (proto.Message, error))
	}
	methodName := proto.MessageName(ReqP(new(Req)))
	_, exists := router.methodsMap[methodName]
	if exists {
		router.err = fmt.Errorf("method %s already registered", methodName)
	}

	interfaceHandler := makeProtoHandler[Req, Resp, ReqP, RespP](handler)
	router.methodsMap[methodName] = interfaceHandler
}

func RegisterInitHandler[Req, Resp any, ReqP Msg[Req], RespP Msg[Resp]](router *InitRouter, handler func(ctx context.Context, msg *Req) (*Resp, error)) {
	router.init = makeProtoHandler[Req, Resp, ReqP, RespP](handler)
}

type InitRouter struct {
	init func(ctx context.Context, msg proto.Message) (proto.Message, error)
}

func (i InitRouter) Handler() func(ctx context.Context, msg proto.Message) (proto.Message, error) {
	if i.init == nil {
		RegisterInitHandler(&i, func(ctx context.Context, msg *emptypb.Empty) (*emptypb.Empty, error) {
			return &emptypb.Empty{}, nil
		})
	}
	return i.init
}

func makeProtoHandler[
	Req, Resp any, ReqP Msg[Req], RespP Msg[Resp],
](
	handler func(ctx context.Context, req *Req) (*Resp, error),
) func(context.Context, proto.Message) (proto.Message, error) {
	reqName := proto.MessageName(ReqP(new(Req)))
	return func(ctx context.Context, req proto.Message) (proto.Message, error) {
		concreteRequest, ok := req.(ReqP)
		if !ok {
			invalidReqName := proto.MessageName(req)
			return nil, fmt.Errorf("expected request of type %s, got %s", reqName, invalidReqName)
		}

		resp, err := handler(ctx, (*Req)(concreteRequest))
		if err != nil {
			return nil, err
		}
		return RespP(resp), nil
	}
}