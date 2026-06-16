package ws

import (
	"log/slog"
	"net/http"

	"goserveractorfsm/internal/actor"
	"goserveractorfsm/internal/game"
	"goserveractorfsm/internal/gamelogic"
	"goserveractorfsm/internal/logx"
	"goserveractorfsm/internal/protocol"
	kickpb "goserveractorfsm/internal/protocol/pb"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

type Handler struct {
	upgrader websocket.Upgrader
	logger   *slog.Logger
}

func NewHandler() http.Handler {
	return NewHandlerWithLogger(logx.Default())
}

func NewHandlerWithLogger(logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = logx.Default()
	}

	return &Handler{
		logger: logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/ws" {
		http.NotFound(w, r)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	remote := conn.RemoteAddr().String()
	h.logger.Info("client_connected", slog.String("remote", remote))
	defer h.logger.Info("client_disconnected", slog.String("remote", remote))

	player := actor.Start(game.NewPlayerActor())
	defer player.Stop()

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if messageType != websocket.BinaryMessage {
			continue
		}

		response, err := handleBinaryMessage(player, h.logger, data)
		if err != nil {
			h.logger.Error("message_failed", slog.String("remote", remote), slog.String("error", err.Error()))
			return
		}
		if err := conn.WriteMessage(websocket.BinaryMessage, response); err != nil {
			h.logger.Error("message_write_failed", slog.String("remote", remote), slog.String("error", err.Error()))
			return
		}
	}
}

func handleBinaryMessage(player *actor.ActorRef, logger *slog.Logger, data []byte) ([]byte, error) {
	envelope := &kickpb.Envelope{}
	if err := proto.Unmarshal(data, envelope); err != nil {
		return nil, err
	}

	logger.Info("message_received",
		slog.Int("msg_id", int(envelope.GetMsgId())),
		slog.String("msg_name", protocol.MsgName(protocol.MsgID(envelope.GetMsgId()))),
		slog.Int("seq_id", int(envelope.GetSeqId())),
		slog.Int("payload_bytes", len(envelope.GetPayload())),
	)

	switch protocol.MsgID(envelope.GetMsgId()) {
	case protocol.KickReqID:
		return handleKick(player, logger, envelope)
	default:
		return encodeError(envelope.GetSeqId(), "unknown command")
	}
}

func handleKick(player *actor.ActorRef, logger *slog.Logger, envelope *kickpb.Envelope) ([]byte, error) {
	seqID := envelope.GetSeqId()
	request := &kickpb.KickRequest{}
	if err := proto.Unmarshal(envelope.GetPayload(), request); err != nil {
		return nil, err
	}

	logger.Info("kick_request",
		slog.Int("seq_id", int(seqID)),
		slog.Int("hammer_type", int(request.GetHammerType())),
		slog.Int("shrew_count", len(request.GetShrews())),
		slog.Int("combo_id", int(request.GetComboId())),
	)

	reply := make(chan game.KickReply, 1)
	if err := player.Tell(game.KickCommand{
		Input: toKickInput(request),
		Reply: reply,
	}); err != nil {
		return nil, err
	}

	result := <-reply
	if result.Err != nil {
		return encodeError(seqID, result.Err.Error())
	}

	response := toKickResponse(request, result.Result)
	logger.Info("kick_response",
		slog.Int("seq_id", int(seqID)),
		slog.Int("ret", int(response.GetRet())),
		slog.Int("money", int(response.GetMoney())),
		slog.Int("angry", int(response.GetAngry())),
		slog.Int("power", int(response.GetPower())),
		slog.Int("level_score", int(response.GetLevelScore())),
		slog.Int("num_of_shrew", int(response.GetNumOfShrew())),
		slog.Int("combo", int(response.GetCombo())),
		slog.Int("combo_id", int(response.GetComboId())),
	)

	return encodeEnvelope(protocol.KickRespID, seqID, response)
}

func toKickInput(request *kickpb.KickRequest) gamelogic.KickInput {
	shrews := make([]gamelogic.KickShrew, 0, len(request.GetShrews()))
	for _, shrew := range request.GetShrews() {
		shrews = append(shrews, gamelogic.KickShrew{
			Index:       int(shrew.GetShrewIndex()),
			ProtectType: int(shrew.GetProtectType()),
		})
	}

	return gamelogic.KickInput{
		HammerType: int(request.GetHammerType()),
		ComboID:    int(request.GetComboId()),
		Shrews:     shrews,
	}
}

func toKickResponse(request *kickpb.KickRequest, result gamelogic.KickResult) *kickpb.KickResponse {
	rewards := make([]*kickpb.ShrewReward, 0, len(result.ShrewRewards))
	for _, reward := range result.ShrewRewards {
		rewards = append(rewards, &kickpb.ShrewReward{
			ShrewIndex: int32(reward.ShrewIndex),
			Reward:     int32(reward.Reward),
		})
	}

	return &kickpb.KickResponse{
		Ret:        0,
		Money:      int32(result.Money),
		Angry:      int32(result.Angry),
		Power:      int32(result.Power),
		LevelScore: int32(result.LevelScore),
		HammerId:   int32(result.HammerID),
		NumOfShrew: int32(result.NumOfShrew),
		ShrewResp:  rewards,
		Combo:      int32(result.Combo),
		ComboId:    int32(result.ComboID),
	}
}

func encodeError(seqID uint32, message string) ([]byte, error) {
	return encodeEnvelope(protocol.ErrorRespID, seqID, &kickpb.ErrorResponse{
		Code:    -1,
		Message: message,
	})
}

func encodeEnvelope(msgID protocol.MsgID, seqID uint32, payload proto.Message) ([]byte, error) {
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
