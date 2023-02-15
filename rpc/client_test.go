package rpc

import (
	"context"
	"errors"
	"myhomework/rpc/message"
	"myhomework/rpc/serialize/json"
	"testing"

	"myhomework/rpc/compression/zstd"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	_ "google.golang.org/grpc/resolver"
)

func Test_setFuncField(t *testing.T) {
	testCases := []struct {
		name string

		mock func(ctrl *gomock.Controller) Proxy

		service Service
		wantErr error
	}{
		{
			name:    "nil",
			service: nil,
			mock: func(ctrl *gomock.Controller) Proxy {
				return NewMockProxy(ctrl)
			},
			wantErr: errors.New("rpc: 不支持 nil"),
		},
		{
			name:    "no pointer",
			service: UserService{},
			mock: func(ctrl *gomock.Controller) Proxy {
				return NewMockProxy(ctrl)
			},
			wantErr: errors.New("rpc: 只支持指向结构体的一级指针"),
		},
		{
			name: "user service",
			mock: func(ctrl *gomock.Controller) Proxy {
				p := NewMockProxy(ctrl)
				p.EXPECT().Invoke(gomock.Any(), &message.Request{
					HeadLength:  36,
					BodyLength:  10,
					Serializer:  1,
					Meta:        make(map[string]string, 2),
					ServiceName: "user-service",
					MethodName:  "GetById",
					Data:        []byte(`{"Id":123}`),
				}).Return(&message.Response{}, nil)
				return p
			},
			service: &UserService{},
		},
	}
	s := &json.Serializer{}
	c := &zstd.Compressor{}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			err := setFuncField(tc.service, tc.mock(ctrl), s, c)
			assert.Equal(t, tc.wantErr, err)
			if err != nil {
				return
			}
			resp, err := tc.service.(*UserService).GetById(context.Background(), &GetByIdReq{Id: 123})
			assert.Equal(t, tc.wantErr, err)
			t.Log(resp)
		})
	}
}
