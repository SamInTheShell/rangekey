// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v5.27.1
// source: api/rangedb/v1/rangedb.proto

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// RangeDBClient is the client API for RangeDB service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type RangeDBClient interface {
	// Core key-value operations
	Get(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (*GetResponse, error)
	Put(ctx context.Context, in *PutRequest, opts ...grpc.CallOption) (*PutResponse, error)
	Delete(ctx context.Context, in *DeleteRequest, opts ...grpc.CallOption) (*DeleteResponse, error)
	Range(ctx context.Context, in *RangeRequest, opts ...grpc.CallOption) (RangeDB_RangeClient, error)
	// Batch operations
	Batch(ctx context.Context, in *BatchRequest, opts ...grpc.CallOption) (*BatchResponse, error)
	// Transaction operations
	BeginTransaction(ctx context.Context, in *BeginTransactionRequest, opts ...grpc.CallOption) (*BeginTransactionResponse, error)
	CommitTransaction(ctx context.Context, in *CommitTransactionRequest, opts ...grpc.CallOption) (*CommitTransactionResponse, error)
	RollbackTransaction(ctx context.Context, in *RollbackTransactionRequest, opts ...grpc.CallOption) (*RollbackTransactionResponse, error)
	// Cluster operations
	GetClusterInfo(ctx context.Context, in *GetClusterInfoRequest, opts ...grpc.CallOption) (*GetClusterInfoResponse, error)
	GetNodeInfo(ctx context.Context, in *GetNodeInfoRequest, opts ...grpc.CallOption) (*GetNodeInfoResponse, error)
	JoinCluster(ctx context.Context, in *JoinClusterRequest, opts ...grpc.CallOption) (*JoinClusterResponse, error)
	// Backup and restore operations
	CreateBackup(ctx context.Context, in *CreateBackupRequest, opts ...grpc.CallOption) (*CreateBackupResponse, error)
	RestoreBackup(ctx context.Context, in *RestoreBackupRequest, opts ...grpc.CallOption) (*RestoreBackupResponse, error)
	GetBackupMetadata(ctx context.Context, in *GetBackupMetadataRequest, opts ...grpc.CallOption) (*GetBackupMetadataResponse, error)
}

type rangeDBClient struct {
	cc grpc.ClientConnInterface
}

func NewRangeDBClient(cc grpc.ClientConnInterface) RangeDBClient {
	return &rangeDBClient{cc}
}

func (c *rangeDBClient) Get(ctx context.Context, in *GetRequest, opts ...grpc.CallOption) (*GetResponse, error) {
	out := new(GetResponse)
	err := c.cc.Invoke(ctx, "/rangedb.v1.RangeDB/Get", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rangeDBClient) Put(ctx context.Context, in *PutRequest, opts ...grpc.CallOption) (*PutResponse, error) {
	out := new(PutResponse)
	err := c.cc.Invoke(ctx, "/rangedb.v1.RangeDB/Put", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rangeDBClient) Delete(ctx context.Context, in *DeleteRequest, opts ...grpc.CallOption) (*DeleteResponse, error) {
	out := new(DeleteResponse)
	err := c.cc.Invoke(ctx, "/rangedb.v1.RangeDB/Delete", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rangeDBClient) Range(ctx context.Context, in *RangeRequest, opts ...grpc.CallOption) (RangeDB_RangeClient, error) {
	stream, err := c.cc.NewStream(ctx, &RangeDB_ServiceDesc.Streams[0], "/rangedb.v1.RangeDB/Range", opts...)
	if err != nil {
		return nil, err
	}
	x := &rangeDBRangeClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type RangeDB_RangeClient interface {
	Recv() (*RangeResponse, error)
	grpc.ClientStream
}

type rangeDBRangeClient struct {
	grpc.ClientStream
}

func (x *rangeDBRangeClient) Recv() (*RangeResponse, error) {
	m := new(RangeResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *rangeDBClient) Batch(ctx context.Context, in *BatchRequest, opts ...grpc.CallOption) (*BatchResponse, error) {
	out := new(BatchResponse)
	err := c.cc.Invoke(ctx, "/rangedb.v1.RangeDB/Batch", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rangeDBClient) BeginTransaction(ctx context.Context, in *BeginTransactionRequest, opts ...grpc.CallOption) (*BeginTransactionResponse, error) {
	out := new(BeginTransactionResponse)
	err := c.cc.Invoke(ctx, "/rangedb.v1.RangeDB/BeginTransaction", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rangeDBClient) CommitTransaction(ctx context.Context, in *CommitTransactionRequest, opts ...grpc.CallOption) (*CommitTransactionResponse, error) {
	out := new(CommitTransactionResponse)
	err := c.cc.Invoke(ctx, "/rangedb.v1.RangeDB/CommitTransaction", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rangeDBClient) RollbackTransaction(ctx context.Context, in *RollbackTransactionRequest, opts ...grpc.CallOption) (*RollbackTransactionResponse, error) {
	out := new(RollbackTransactionResponse)
	err := c.cc.Invoke(ctx, "/rangedb.v1.RangeDB/RollbackTransaction", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rangeDBClient) GetClusterInfo(ctx context.Context, in *GetClusterInfoRequest, opts ...grpc.CallOption) (*GetClusterInfoResponse, error) {
	out := new(GetClusterInfoResponse)
	err := c.cc.Invoke(ctx, "/rangedb.v1.RangeDB/GetClusterInfo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rangeDBClient) GetNodeInfo(ctx context.Context, in *GetNodeInfoRequest, opts ...grpc.CallOption) (*GetNodeInfoResponse, error) {
	out := new(GetNodeInfoResponse)
	err := c.cc.Invoke(ctx, "/rangedb.v1.RangeDB/GetNodeInfo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rangeDBClient) JoinCluster(ctx context.Context, in *JoinClusterRequest, opts ...grpc.CallOption) (*JoinClusterResponse, error) {
	out := new(JoinClusterResponse)
	err := c.cc.Invoke(ctx, "/rangedb.v1.RangeDB/JoinCluster", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rangeDBClient) CreateBackup(ctx context.Context, in *CreateBackupRequest, opts ...grpc.CallOption) (*CreateBackupResponse, error) {
	out := new(CreateBackupResponse)
	err := c.cc.Invoke(ctx, "/rangedb.v1.RangeDB/CreateBackup", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rangeDBClient) RestoreBackup(ctx context.Context, in *RestoreBackupRequest, opts ...grpc.CallOption) (*RestoreBackupResponse, error) {
	out := new(RestoreBackupResponse)
	err := c.cc.Invoke(ctx, "/rangedb.v1.RangeDB/RestoreBackup", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *rangeDBClient) GetBackupMetadata(ctx context.Context, in *GetBackupMetadataRequest, opts ...grpc.CallOption) (*GetBackupMetadataResponse, error) {
	out := new(GetBackupMetadataResponse)
	err := c.cc.Invoke(ctx, "/rangedb.v1.RangeDB/GetBackupMetadata", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RangeDBServer is the server API for RangeDB service.
// All implementations must embed UnimplementedRangeDBServer
// for forward compatibility
type RangeDBServer interface {
	// Core key-value operations
	Get(context.Context, *GetRequest) (*GetResponse, error)
	Put(context.Context, *PutRequest) (*PutResponse, error)
	Delete(context.Context, *DeleteRequest) (*DeleteResponse, error)
	Range(*RangeRequest, RangeDB_RangeServer) error
	// Batch operations
	Batch(context.Context, *BatchRequest) (*BatchResponse, error)
	// Transaction operations
	BeginTransaction(context.Context, *BeginTransactionRequest) (*BeginTransactionResponse, error)
	CommitTransaction(context.Context, *CommitTransactionRequest) (*CommitTransactionResponse, error)
	RollbackTransaction(context.Context, *RollbackTransactionRequest) (*RollbackTransactionResponse, error)
	// Cluster operations
	GetClusterInfo(context.Context, *GetClusterInfoRequest) (*GetClusterInfoResponse, error)
	GetNodeInfo(context.Context, *GetNodeInfoRequest) (*GetNodeInfoResponse, error)
	JoinCluster(context.Context, *JoinClusterRequest) (*JoinClusterResponse, error)
	// Backup and restore operations
	CreateBackup(context.Context, *CreateBackupRequest) (*CreateBackupResponse, error)
	RestoreBackup(context.Context, *RestoreBackupRequest) (*RestoreBackupResponse, error)
	GetBackupMetadata(context.Context, *GetBackupMetadataRequest) (*GetBackupMetadataResponse, error)
	mustEmbedUnimplementedRangeDBServer()
}

// UnimplementedRangeDBServer must be embedded to have forward compatible implementations.
type UnimplementedRangeDBServer struct {
}

func (UnimplementedRangeDBServer) Get(context.Context, *GetRequest) (*GetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Get not implemented")
}
func (UnimplementedRangeDBServer) Put(context.Context, *PutRequest) (*PutResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Put not implemented")
}
func (UnimplementedRangeDBServer) Delete(context.Context, *DeleteRequest) (*DeleteResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedRangeDBServer) Range(*RangeRequest, RangeDB_RangeServer) error {
	return status.Errorf(codes.Unimplemented, "method Range not implemented")
}
func (UnimplementedRangeDBServer) Batch(context.Context, *BatchRequest) (*BatchResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Batch not implemented")
}
func (UnimplementedRangeDBServer) BeginTransaction(context.Context, *BeginTransactionRequest) (*BeginTransactionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BeginTransaction not implemented")
}
func (UnimplementedRangeDBServer) CommitTransaction(context.Context, *CommitTransactionRequest) (*CommitTransactionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CommitTransaction not implemented")
}
func (UnimplementedRangeDBServer) RollbackTransaction(context.Context, *RollbackTransactionRequest) (*RollbackTransactionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RollbackTransaction not implemented")
}
func (UnimplementedRangeDBServer) GetClusterInfo(context.Context, *GetClusterInfoRequest) (*GetClusterInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetClusterInfo not implemented")
}
func (UnimplementedRangeDBServer) GetNodeInfo(context.Context, *GetNodeInfoRequest) (*GetNodeInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetNodeInfo not implemented")
}
func (UnimplementedRangeDBServer) JoinCluster(context.Context, *JoinClusterRequest) (*JoinClusterResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method JoinCluster not implemented")
}
func (UnimplementedRangeDBServer) CreateBackup(context.Context, *CreateBackupRequest) (*CreateBackupResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateBackup not implemented")
}
func (UnimplementedRangeDBServer) RestoreBackup(context.Context, *RestoreBackupRequest) (*RestoreBackupResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RestoreBackup not implemented")
}
func (UnimplementedRangeDBServer) GetBackupMetadata(context.Context, *GetBackupMetadataRequest) (*GetBackupMetadataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBackupMetadata not implemented")
}
func (UnimplementedRangeDBServer) mustEmbedUnimplementedRangeDBServer() {}

// UnsafeRangeDBServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to RangeDBServer will
// result in compilation errors.
type UnsafeRangeDBServer interface {
	mustEmbedUnimplementedRangeDBServer()
}

func RegisterRangeDBServer(s grpc.ServiceRegistrar, srv RangeDBServer) {
	s.RegisterService(&RangeDB_ServiceDesc, srv)
}

func _RangeDB_Get_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RangeDBServer).Get(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rangedb.v1.RangeDB/Get",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RangeDBServer).Get(ctx, req.(*GetRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RangeDB_Put_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PutRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RangeDBServer).Put(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rangedb.v1.RangeDB/Put",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RangeDBServer).Put(ctx, req.(*PutRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RangeDB_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeleteRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RangeDBServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rangedb.v1.RangeDB/Delete",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RangeDBServer).Delete(ctx, req.(*DeleteRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RangeDB_Range_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(RangeRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(RangeDBServer).Range(m, &rangeDBRangeServer{stream})
}

type RangeDB_RangeServer interface {
	Send(*RangeResponse) error
	grpc.ServerStream
}

type rangeDBRangeServer struct {
	grpc.ServerStream
}

func (x *rangeDBRangeServer) Send(m *RangeResponse) error {
	return x.ServerStream.SendMsg(m)
}

func _RangeDB_Batch_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BatchRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RangeDBServer).Batch(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rangedb.v1.RangeDB/Batch",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RangeDBServer).Batch(ctx, req.(*BatchRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RangeDB_BeginTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BeginTransactionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RangeDBServer).BeginTransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rangedb.v1.RangeDB/BeginTransaction",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RangeDBServer).BeginTransaction(ctx, req.(*BeginTransactionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RangeDB_CommitTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CommitTransactionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RangeDBServer).CommitTransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rangedb.v1.RangeDB/CommitTransaction",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RangeDBServer).CommitTransaction(ctx, req.(*CommitTransactionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RangeDB_RollbackTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RollbackTransactionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RangeDBServer).RollbackTransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rangedb.v1.RangeDB/RollbackTransaction",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RangeDBServer).RollbackTransaction(ctx, req.(*RollbackTransactionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RangeDB_GetClusterInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetClusterInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RangeDBServer).GetClusterInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rangedb.v1.RangeDB/GetClusterInfo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RangeDBServer).GetClusterInfo(ctx, req.(*GetClusterInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RangeDB_GetNodeInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetNodeInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RangeDBServer).GetNodeInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rangedb.v1.RangeDB/GetNodeInfo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RangeDBServer).GetNodeInfo(ctx, req.(*GetNodeInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RangeDB_JoinCluster_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(JoinClusterRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RangeDBServer).JoinCluster(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rangedb.v1.RangeDB/JoinCluster",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RangeDBServer).JoinCluster(ctx, req.(*JoinClusterRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RangeDB_CreateBackup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateBackupRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RangeDBServer).CreateBackup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rangedb.v1.RangeDB/CreateBackup",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RangeDBServer).CreateBackup(ctx, req.(*CreateBackupRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RangeDB_RestoreBackup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RestoreBackupRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RangeDBServer).RestoreBackup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rangedb.v1.RangeDB/RestoreBackup",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RangeDBServer).RestoreBackup(ctx, req.(*RestoreBackupRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _RangeDB_GetBackupMetadata_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetBackupMetadataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(RangeDBServer).GetBackupMetadata(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/rangedb.v1.RangeDB/GetBackupMetadata",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(RangeDBServer).GetBackupMetadata(ctx, req.(*GetBackupMetadataRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// RangeDB_ServiceDesc is the grpc.ServiceDesc for RangeDB service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var RangeDB_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "rangedb.v1.RangeDB",
	HandlerType: (*RangeDBServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Get",
			Handler:    _RangeDB_Get_Handler,
		},
		{
			MethodName: "Put",
			Handler:    _RangeDB_Put_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _RangeDB_Delete_Handler,
		},
		{
			MethodName: "Batch",
			Handler:    _RangeDB_Batch_Handler,
		},
		{
			MethodName: "BeginTransaction",
			Handler:    _RangeDB_BeginTransaction_Handler,
		},
		{
			MethodName: "CommitTransaction",
			Handler:    _RangeDB_CommitTransaction_Handler,
		},
		{
			MethodName: "RollbackTransaction",
			Handler:    _RangeDB_RollbackTransaction_Handler,
		},
		{
			MethodName: "GetClusterInfo",
			Handler:    _RangeDB_GetClusterInfo_Handler,
		},
		{
			MethodName: "GetNodeInfo",
			Handler:    _RangeDB_GetNodeInfo_Handler,
		},
		{
			MethodName: "JoinCluster",
			Handler:    _RangeDB_JoinCluster_Handler,
		},
		{
			MethodName: "CreateBackup",
			Handler:    _RangeDB_CreateBackup_Handler,
		},
		{
			MethodName: "RestoreBackup",
			Handler:    _RangeDB_RestoreBackup_Handler,
		},
		{
			MethodName: "GetBackupMetadata",
			Handler:    _RangeDB_GetBackupMetadata_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Range",
			Handler:       _RangeDB_Range_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "api/rangedb/v1/rangedb.proto",
}
