package protocol

import (
	kickpb "goserveractorfsm/internal/protocol/pb"

	"google.golang.org/protobuf/proto"
)

const (
	PingReqID           MsgID = 1
	PongRespID          MsgID = 2
	JoinRoomReqID       MsgID = 1001
	JoinRoomRespID      MsgID = 1002
	GameSnapshotReqID   MsgID = 1003
	GameSnapshotRespID  MsgID = 1004
	TimeSyncReqID       MsgID = 1005
	TimeSyncRespID      MsgID = 1006
	KickReqID           MsgID = 2001
	KickRespID          MsgID = 2002
	ShrewTimelinePushID MsgID = 3001
	ShrewStatePushID    MsgID = 3002
	ErrorRespID         MsgID = 9001
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
	case JoinRoomReqID:
		return "join_room_req"
	case JoinRoomRespID:
		return "join_room_resp"
	case GameSnapshotReqID:
		return "game_snapshot_req"
	case GameSnapshotRespID:
		return "game_snapshot_resp"
	case TimeSyncReqID:
		return "time_sync_req"
	case TimeSyncRespID:
		return "time_sync_resp"
	case KickReqID:
		return "kick_req"
	case KickRespID:
		return "kick_resp"
	case ShrewTimelinePushID:
		return "shrew_timeline_push"
	case ShrewStatePushID:
		return "shrew_state_push"
	case ErrorRespID:
		return "error_resp"
	default:
		return "unknown"
	}
}

func Encode(message proto.Message) ([]byte, error) {
	return proto.Marshal(message)
}

func Decode(data []byte, message proto.Message) error {
	return proto.Unmarshal(data, message)
}

func EncodeEnvelope(msgID MsgID, seqID uint32, payload proto.Message) ([]byte, error) {
	payloadData, err := proto.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return proto.Marshal(&kickpb.Envelope{
		SeqId:   seqID,
		MsgId:   msgID.Uint32(),
		Payload: payloadData,
	})
}

func DecodeEnvelope(data []byte) (*kickpb.Envelope, error) {
	envelope := &kickpb.Envelope{}
	if err := proto.Unmarshal(data, envelope); err != nil {
		return nil, err
	}
	return envelope, nil
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
