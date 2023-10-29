package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	hertzGzip "github.com/hertz-contrib/gzip"

	"github.com/davexpro/hertz-binding-issue/pb_gen"
)

const (
	testUri = "http://localhost:8888/example"
)

func main() {
	go func() {
		time.Sleep(time.Second * 3)
		postExample()
	}()
	h := server.Default()
	h.Use(hertzGzip.Gzip(hertzGzip.DefaultCompression, hertzGzip.WithDecompressFn(hertzGzip.DefaultDecompressHandle)))
	h.POST("/example", handleExample)
	h.Spin()
}

func handleExample(ctx context.Context, reqCtx *app.RequestContext) {
	req := &pb_gen.ExampleRequest{}
	err := reqCtx.Bind(req)
	if err != nil {
		panic(err)
	}
	str, _ := sonic.MarshalString(req)
	log.Printf("[srv] req: %s", str)
	if (req.Alpha) == "" {
		log.Printf("!!! request is empty")
	}
	resp := &pb_gen.ExampleResponse{
		Foxtrot: "Foxtrot",
		Golf:    "Golf",
		Hotel:   "Hotel",
		India:   time.Now().Unix(),
	}
	reqCtx.JSON(http.StatusOK, resp)
}

func postExample() {
	// 1. marshal request to protobuf
	ctx := context.Background()
	req := &pb_gen.ExampleRequest{
		Alpha:   "pi=3.1415926",
		Bravo:   "e=2.718281828",
		Charlie: "g=9.8",
		Delta:   "c=299792458",
		Echo:    31415926,
	}

	protoBs := make([]byte, req.Size())
	n := req.FastWrite(protoBs)
	log.Printf("pb bytes: %d (ori) => %d", req.Size(), n)

	// 2. gzip request
	buf := bytes.NewBuffer(make([]byte, 0, 512))
	gw, _ := gzip.NewWriterLevel(buf, 5)
	if _, err := gw.Write(protoBs); err != nil {
		log.Printf("gzip `Write` failed, detail: %s", err)
		panic(err)
	}
	gw.Close()
	log.Printf("gzip bytes: %d (ori) => %d", n, buf.Len())

	// 3. do request
	//httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, testUri, bytes.NewBuffer(protoBs)) // work
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, testUri, buf) // 不 work
	if err != nil {
		panic(err)
	}

	// 4. request use pb
	httpReq.Header.Add("Content-Encoding", "gzip") // 注释这个 gzip 用 raw pb bytes 可以 work
	httpReq.Header.Set("Content-Type", "application/x-protobuf")
	httpReq.Header.Set("User-Agent", "Hertz Test")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	log.Printf("[cli] resp code: %d", resp.StatusCode)
	respBs, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	log.Printf("[cli] resp body: %s", respBs)
}
