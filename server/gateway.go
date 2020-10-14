package server

import (
	"context"
	"log"
	"net/http"
	"strings"

	pb "go-kunpeng/api"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
)

func ProvideHttp(endpoint string, grpcServer *grpc.Server) *http.Server {
	ctx := context.Background()
	gwmux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}))
	dopts := []grpc.DialOption{grpc.WithInsecure()}

	var err error
	err = pb.RegisterCacheUserHandlerFromEndpoint(ctx, gwmux, endpoint, dopts)
	if err != nil {
		log.Printf("Register Endpoint err: %v", err)
	}
	err = pb.RegisterCacheActivityRecordHandlerFromEndpoint(ctx, gwmux, endpoint, dopts)
	if err != nil {
		log.Printf("Register Endpoint err: %v", err)
	}

	// 新建mux，它是http的请求复用器
	mux := http.NewServeMux()
	// 注册gwmux
	mux.Handle("/", gwmux)
	log.Println(endpoint + " HTTP.Listing...")
	return &http.Server{
		Addr:      endpoint,
		Handler:   grpcHandlerFunc(grpcServer, mux),
		TLSConfig: nil,
	}
}

// grpcHandlerFunc 根据不同的请求重定向到指定的Handler处理
func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.Header.Get("X-Real-IP")
		if ip == "" {
			// 当请求头不存在即不存在代理时直接获取ip
			ip = strings.Split(r.RemoteAddr, ":")[0]
		}
		log.Printf("Access from %s with %s to %s %s", ip, r.Header.Get("Content-Type"), r.Method, r.RequestURI)
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	}), &http2.Server{})
}
