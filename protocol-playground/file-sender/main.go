package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/k-code-yt/go-api-practice/protocol-playground/shared"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type FileClient struct {
	client shared.StreamingTransportSerivceClient
}

func (fc *FileClient) UploadFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("couldn't open file %v", err)
	}

	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("err reading filestat %v", err)

	}

	filename := fileInfo.Name()
	filesize := fileInfo.Size()

	stream, err := fc.client.UploadFile(context.Background())
	if err != nil {
		return fmt.Errorf("err uploading via GRPC %v", err)

	}

	buffer := make([]byte, 64)
	var (
		chunkIndex int64
	)

	for {
		n, err := file.Read(buffer)

		if err == io.EOF {
			logrus.Info("Finished file read")
			break
		}

		if err != nil {
			return fmt.Errorf("err reading file %v", err)
		}

		chunk := &shared.FileChunk{
			Filename:   filename,
			Content:    buffer[:n],
			ChunkIndex: chunkIndex,
		}

		if chunkIndex == 0 {
			chunk.TotalSize = filesize
		}

		err = stream.Send(chunk)
		if err != nil {
			return fmt.Errorf("failed to send chunk GRPC %v", err)
		}
		chunkIndex++

	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("GRPC file streaming server failed %v", err)
	}

	if resp.BytesReceived == filesize {
		logrus.Info("file size matches, total success")
	} else {
		logrus.Error("final file size does not match")
	}

	return nil
}

func main() {
	conn, err := grpc.NewClient(fmt.Sprintf("localhost%s", shared.HTTPPortInvoice), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	client := &FileClient{
		client: shared.NewStreamingTransportSerivceClient(conn),
	}

	err = client.UploadFile("./random-xlsx.xlsx")
	if err != nil {
		log.Fatal(err)
	}
}
