package sdkrpc

import (
	context "context"
	"net"
	"os"
	reflect "reflect"

	"github.com/StarfishProgram/starfish-sdk/sdk"
	"github.com/StarfishProgram/starfish-sdk/sdkcodes"
	"github.com/StarfishProgram/starfish-sdk/sdklog"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	anypb "google.golang.org/protobuf/types/known/anypb"
)

type _Server struct {
	UnimplementedGRPCServiceServer
	calls map[string]func(*anypb.Any) *anypb.Any
}

// ServerRegisterCall 注册服务
func ServerRegisterCall[P, R protoreflect.ProtoMessage](server *_Server, call func(param P) R) {
	var p P
	paramAny, err := anypb.New(p)
	if err != nil {
		sdklog.Ins().AddCallerSkip(1).Panic(err)
	}
	pt := reflect.TypeOf(p).Elem()
	server.calls[paramAny.TypeUrl] = func(param *anypb.Any) *anypb.Any {
		realParam := reflect.New(pt).Interface().(P)
		err := param.UnmarshalTo(realParam)
		sdk.CheckError(err, sdkcodes.Internal.WithMsg("%s", err.Error()))
		callResult := call(realParam)
		resultData, err := anypb.New(callResult)
		sdk.CheckError(err, sdkcodes.Internal.WithMsg("%s", err.Error()))
		return resultData
	}
}

func (s *_Server) Call(ctx context.Context, param *anypb.Any) (result *Result, err error) {
	result = &Result{Code: nil, Data: nil}
	call, ok := s.calls[param.TypeUrl]
	if !ok {
		result.Code = &Code{
			Code: sdkcodes.ParamInvalid.Code(),
			Msg:  sdkcodes.ParamInvalid.Msg(),
			I18N: sdkcodes.ParamInvalid.I18n(),
		}
		return
	}
	defer func() {
		if err := recover(); err != nil {
			result.Data = nil
			if code, ok := err.(sdkcodes.Code); ok {
				result.Code = &Code{
					Code: code.Code(),
					Msg:  code.Msg(),
					I18N: code.I18n(),
				}

				sdklog.Ins().Warn(code)
				return
			}
			sdklog.Ins().Error(err)
			result.Code = &Code{
				Code: sdkcodes.Internal.Code(),
				Msg:  sdkcodes.Internal.Msg(),
				I18N: sdkcodes.Internal.I18n(),
			}
		}
	}()
	result.Data = call(param)
	return
}

func InitServer(listener string) (*_Server, chan os.Signal) {
	lis, err := net.Listen("tcp", listener)
	if err != nil {
		sdklog.Ins().Panicf("GRPC服务创建失败 : %s", err.Error())
	}
	server := _Server{
		calls: map[string]func(*anypb.Any) *anypb.Any{},
	}
	rpcServer := grpc.NewServer()
	RegisterGRPCServiceServer(rpcServer, &server)
	ch := make(chan os.Signal, 1)
	go func() {
		if err := rpcServer.Serve(lis); err != nil {
			sdklog.Ins().Error("GRPC服务运行异常", err)
		}
		sdklog.Ins().Info("GRPC服务已停止")
		close(ch)
	}()
	go func() {
		<-ch
		rpcServer.Stop()
	}()
	return &server, ch
}

var clientIns map[string]*_Client

func init() {
	clientIns = make(map[string]*_Client)
}

type _Client struct {
	client GRPCServiceClient
}

func InitClient(url string, key ...string) {
	conn, err := grpc.Dial(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		sdklog.Ins().AddCallerSkip(1).Panic(err)
	}
	client := NewGRPCServiceClient(conn)
	ins := _Client{client: client}
	if len(key) == 0 {
		clientIns[""] = &ins
	} else {
		clientIns[key[0]] = &ins
	}
}

type ClientCallResult[D protoreflect.ProtoMessage] struct {
	Code *Code
	Data D
}

func ClientCall[P, R protoreflect.ProtoMessage](client *_Client, param P) ClientCallResult[R] {
	var r R
	anyParam, err := anypb.New(param)
	if err != nil {
		return ClientCallResult[R]{
			Code: &Code{
				Code: sdkcodes.Internal.Code(),
				Msg:  sdkcodes.Internal.Msg(),
				I18N: sdkcodes.Internal.I18n(),
			},
			Data: r,
		}
	}
	result, err := client.client.Call(sdk.Context(), anyParam)
	if err != nil {
		sdklog.Ins().AddCallerSkip(1).Error(err)
		return ClientCallResult[R]{Code: &Code{
			Code: sdkcodes.Internal.Code(),
			Msg:  sdkcodes.Internal.Msg(),
			I18N: sdkcodes.Internal.I18n(),
		}, Data: r}
	}
	realData := reflect.New(reflect.TypeOf(r).Elem()).Interface().(R)
	if err := result.Data.UnmarshalTo(realData); err != nil {
		return ClientCallResult[R]{Code: &Code{
			Code: sdkcodes.Internal.Code(),
			Msg:  sdkcodes.Internal.Msg(),
			I18N: sdkcodes.Internal.I18n(),
		}, Data: r}
	}
	return ClientCallResult[R]{
		Code: result.Code,
		Data: realData,
	}
}
