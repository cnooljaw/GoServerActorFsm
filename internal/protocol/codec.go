package protocol

import (
	kickpb "goserveractorfsm/internal/protocol/pb"

	"google.golang.org/protobuf/proto"
)

const (
	PingReqID   MsgID = 1
	PongRespID  MsgID = 2
	KickReqID   MsgID = 2001
	KickRespID  MsgID = 2002
	ErrorRespID MsgID = 9001
)

type MsgID uint32

func (id MsgID) Uint32() uint32 {
	return uint32(id)
}

func MsgName(id MsgID) string {
	switch id {
	case PingReqID:
		return "ping_req"
	case PongRespID:
		return "pong_resp"
	case KickReqID:
		return "kick_req"
	case KickRespID:
		return "kick_resp"
	case ErrorRespID:
		return "error_resp"
	default:
		return "unknown"
	}
}

func Encode(message proto.Message) ([]byte, error) {
	return proto.Marshal(message)
}

func DecodeKickRequest(data []byte) (*kickpb.KickRequest, error) {
	request := &kickpb.KickRequest{}
	if err := proto.Unmarshal(data, request); err != nil {
		return nil, err
	}
	return request, nil
}

func DecodeKickResponse(data []byte) (*kickpb.KickResponse, error) {
	response := &kickpb.KickResponse{}
	if err := proto.Unmarshal(data, response); err != nil {
		return nil, err
	}
	return response, nil
}
