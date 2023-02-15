# **需求分析**

## **场景分析**

- 开发者希望在某个方法上启用压缩算法。例如在获取文章摘要的时候并不需要启用压缩算法，但是在获取文章详细内容的时候启用压缩算法

- 开发者希望在某个服务上启用压缩算法

- 开发者希望请求或者响应能够单独控制是否使用压缩算法，以及使用不同的压缩算法：

- - 请求过去不使用压缩算法，但是响应过来使用了压缩算法；反过来也可以
  - 请求过去使用了 A 算法，但是响应过来使用了 B 算法

- 开发者希望能够自动检测传输数据大小，在传输数据超过一个阈值的时候启用压缩算法

- 开发者希望能够实时检测网络负载，在网络负载很高的时候启用压缩算法。跟进一步地，开发者希望在负载低的时候启用低压缩率的算法，在负载高的时候启用高压缩率的算法

- 开发者希望能够实时检测 CPU 使用率，在 CPU 使用率很高的时候弃用压缩算法



虽然在理论上开发者有很多类似的场景，但是我们并不需要提供那么丰富多变的压缩功能。尤其是动态检测部分，即依据系统的实时情况来决定是否要启用压缩，以及启用什么压缩算法。（注：这一类多集中在理论上探讨，实际中确实没有什么系统会为了一个简单的压缩功能考虑这么复杂，但是如果你们面试就可以这么吹一波）

## **功能需求**

- 支持不同的压缩算法
- 允许用户接入自定义算法
- 用户可以在服务层面上控制压缩算法，但是不需要在方法级别上控制压缩与否，以及使用何种算法
- 响应将采用和请求一样的压缩算法。即如果请求采用了压缩，那么响应会启用压缩算法，并且使用同样的压缩算法

## **非功能需求**

- 良好的扩展性，即用户可以使用不同的压缩算法
- 在将来我们可能会在方法级别，以及请求和响应分别控制压缩与否与压缩算法，所以在设计和实现的时候要注意别太死板

# **行业方案**

gin的压缩中间件https://github.com/gin-contrib/gzip/blob/master/gzip.go

1. 定义了Gzip的handler，作为middleware
2. func (g *gzipHandler) Handle(c *gin.Context)的实现中，重新赋值了`Write`、`WriteString`、`WriteHeader`, 上述函数作为go SDK中`compress/gzip`的包装；

# **设计**

### 接口设计

```go
package compression

type Compression interface {
	Code() uint8
	Compress(val []byte) ([]byte, error)
	Decompress(data []byte) ([]byte, error)
}
```

### zstd实现

```go
package zstd

import (
	"github.com/klauspost/compress/zstd"
)

type Compressor struct {
}

func (c *Compressor) Code() uint8 {
	return 1
}

func (c *Compressor) Compress(data []byte) ([]byte, error) {
	var encoder, _ = zstd.NewWriter(nil)
	return encoder.EncodeAll(data, make([]byte, 0, len(data))), nil
}

func (c *Compressor) Decompress(data []byte) ([]byte, error) {
	var decoder, _ = zstd.NewReader(nil, zstd.WithDecoderConcurrency(0))
	return decoder.DecodeAll(data, nil)
}
```



### 服务端

1. server端需要增加compressions字段，类型为`map[uint8]compression.Compression`；
   - 做成map是为了方便其他压缩算法的扩展
2. server需要将该字段传递给`reflectionStub；`
3. 在`reflectionStub`的`invoke`方法中，根据`req.Compresser`字段的值，获得压缩算法实例；
4. 使用压缩算法实例，将`Data`的数据进行解压缩；
5. `call`完成后，将回应数据，进行`压缩`后，返回给客户端；



### 客户端

1. Client端需要增加compression字段，类型为`compression.Compression`；
2. 提供`ClientWithCompressor`方法，方便New的时候进行自定义；
3. `setFuncField`中增加参数` c compression.Compression `；
4. 在赋值给`req.Data`之前，需要进行压缩，否则计算body的长度会有问题；
5. 在发起`Invoke`调用后，如果包含数据，则对数据进行解压，然后再做反序列化；

# **测试**

## **单元测试**

因为本身压缩算法都是依赖于 Go SDK，我们只是提供简单的封装，所以只需要进行简单的测试。



## **集成测试**

```go
func TestInitServiceProto(t *testing.T) {
	server := NewServer()
	service := &UserServiceServer{}
	server.RegisterService(service)
	server.RegisterSerializer(&proto.Serializer{})
	server.RegisterCompression(&zstd.Compressor{})   // 服务端注册 compressor
	go func() {
		err := server.Start("tcp", ":8081")
		t.Log(err)
	}()
	time.Sleep(time.Second * 3)

	usClient := &UserService{}
	client, err := NewClient(":8081", ClientWithSerializer(&proto.Serializer{}), ClientWithCompressor(&zstd.Compressor{})) // 客户端opts，使用zstd
	require.NoError(t, err)
	err = client.InitService(usClient)
	require.NoError(t, err)

	testCases := []struct {
		name string
		mock func()

		wantErr  error
		wantResp *GetByIdResp
	}{
		{
			name: "no error",
			mock: func() {
				service.Err = nil
				service.Msg = "hello, world"
			},
			wantResp: &GetByIdResp{
				Msg: "hello, world",
			},
		},
		{
			name: "error",
			mock: func() {
				service.Msg = ""
				service.Err = errors.New("mock error")
			},
			wantResp: &GetByIdResp{},
			wantErr:  errors.New("mock error"),
		},

		{
			name: "both",
			mock: func() {
				service.Msg = "hello, world"
				service.Err = errors.New("mock error")
			},
			wantResp: &GetByIdResp{
				Msg: "hello, world",
			},
			wantErr: errors.New("mock error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mock()
			resp, er := usClient.GetByIdProto(context.Background(), &gen.GetByIdReq{Id: 123})
			assert.Equal(t, tc.wantErr, er)
			if resp != nil && resp.User != nil {
				assert.Equal(t, tc.wantResp.Msg, resp.User.Name)
			}
		})
	}
}
```



## **模糊测试**

实际上，如果我们自己实现一个压缩算法，那么可以考虑使用模糊测试。即对模糊测试生成的数据进行压缩，再解压缩，应该能够还原为原始数据。

不过在我们的 gzip，是 Go SDK 实现的，所以我们可以不必设计模糊测试。