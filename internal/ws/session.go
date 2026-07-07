package ws

import (
	"log/slog"
	"net/http"
	"sync/atomic"

	"goserveractorfsm/internal/actor"
	"goserveractorfsm/internal/config"
	"goserveractorfsm/internal/logx"
	"goserveractorfsm/internal/protocol"
	kickpb "goserveractorfsm/internal/protocol/pb"
	"goserveractorfsm/internal/room"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

type Handler struct {
	upgrader      websocket.Upgrader
	logger        *slog.Logger
	room          *actor.ActorRef
	nextSessionID atomic.Uint64
}

func NewHandler() http.Handler {
	return NewHandlerWithConfig(config.Default(), logx.Default())
}

func NewHandlerWithLogger(logger *slog.Logger) http.Handler {
	return NewHandlerWithConfig(config.Default(), logger)
}

func NewHandlerWithConfig(cfg config.ServerConfig, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = logx.Default()
	}

	return &Handler{
		logger: logger,
		room: actor.Start(room.NewRoomActor(room.Config{
			RoomSize:  cfg.RoomSize,
			HoleCount: cfg.HoleCount,
		})),
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

	sessionID := h.nextSessionID.Add(1)
	remote := conn.RemoteAddr().String()
	h.logger.Info("client_connected", slog.Uint64("session_id", sessionID), slog.String("remote", remote))
	defer h.logger.Info("client_disconnected", slog.Uint64("session_id", sessionID), slog.String("remote", remote))

	outbound := make(chan []byte, 32)
	done := make(chan struct{})
	joinReply := make(chan room.JoinSessionReply, 1)
	if err := h.room.Tell(room.JoinSession{
		Session: room.SessionRef{SessionID: sessionID, Outbound: outbound},
		Reply:   joinReply,
	}); err != nil {
		h.logger.Error("room_join_failed", slog.Uint64("session_id", sessionID), slog.String("error", err.Error()))
		return
	}
	joined := <-joinReply
	if joined.Err != nil {
		h.logger.Error("room_join_failed", slog.Uint64("session_id", sessionID), slog.String("error", joined.Err.Error()))
		return
	}
	h.logger.Info("room_joined",
		slog.Uint64("session_id", sessionID),
		slog.Uint64("player_id", joined.PlayerID),
		slog.Uint64("attack_id", joined.AttackID),
		slog.Uint64("attack_epoch", joined.AttackEpoch),
		slog.Int("seat_index", joined.SeatIndex),
		slog.Int("room_size", joined.RoomSize),
	)
	defer h.room.Tell(room.LeaveSession{SessionID: sessionID})

	go h.writeLoop(conn, outbound, done, sessionID, remote)
	defer close(done)

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if messageType != websocket.BinaryMessage {
			continue
		}

		envelope, err := protocol.DecodeEnvelope(data)
		if err != nil {
			h.logger.Error("message_failed", slog.Uint64("session_id", sessionID), slog.String("remote", remote), slog.String("error", err.Error()))
			return
		}

		h.logger.Info("message_received",
			slog.Uint64("session_id", sessionID),
			slog.Int("msg_id", int(envelope.GetMsgId())),
			slog.String("msg_name", protocol.MsgName(protocol.MsgID(envelope.GetMsgId()))),
			slog.Int("seq_id", int(envelope.GetSeqId())),
			slog.Int("payload_bytes", len(envelope.GetPayload())),
		)
		h.logInboundDetail(sessionID, envelope)

		if err := h.room.Tell(room.ClientEnvelope{SessionID: sessionID, Envelope: envelope}); err != nil {
			h.logger.Error("message_dispatch_failed", slog.Uint64("session_id", sessionID), slog.String("remote", remote), slog.String("error", err.Error()))
			return
		}
	}
}

func (h *Handler) writeLoop(conn *websocket.Conn, outbound <-chan []byte, done <-chan struct{}, sessionID uint64, remote string) {
	for {
		select {
		case <-done:
			return
		case data := <-outbound:
			envelope := &kickpb.Envelope{}
			if err := proto.Unmarshal(data, envelope); err == nil {
				h.logger.Info("message_send",
					slog.Uint64("session_id", sessionID),
					slog.Int("msg_id", int(envelope.GetMsgId())),
					slog.String("msg_name", protocol.MsgName(protocol.MsgID(envelope.GetMsgId()))),
					slog.Int("seq_id", int(envelope.GetSeqId())),
					slog.Int("payload_bytes", len(envelope.GetPayload())),
				)
				h.logOutboundDetail(sessionID, envelope)
			}
			if err := conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
				h.logger.Error("message_write_failed", slog.Uint64("session_id", sessionID), slog.String("remote", remote), slog.String("error", err.Error()))
				return
			}
		}
	}
}

func (h *Handler) logInboundDetail(sessionID uint64, envelope *kickpb.Envelope) {
	switch protocol.MsgID(envelope.GetMsgId()) {
	case protocol.KickReqID:
		request := &kickpb.KickRequest{}
		if err := proto.Unmarshal(envelope.GetPayload(), request); err != nil {
			return
		}
		h.logger.Info("kick_request",
			slog.Uint64("session_id", sessionID),
			slog.Int("seq_id", int(envelope.GetSeqId())),
			slog.Uint64("attack_epoch", request.GetAttackEpoch()),
			slog.Int("hammer_type", int(request.GetHammerType())),
			slog.Int("shrew_count", len(request.GetShrews())),
			slog.Int("combo_id", int(request.GetComboId())),
		)
	}
}

func (h *Handler) logOutboundDetail(sessionID uint64, envelope *kickpb.Envelope) {
	switch protocol.MsgID(envelope.GetMsgId()) {
	case protocol.KickRespID:
		response := &kickpb.KickResponse{}
		if err := proto.Unmarshal(envelope.GetPayload(), response); err != nil {
			return
		}
		h.logger.Info("kick_response",
			slog.Uint64("session_id", sessionID),
			slog.Int("seq_id", int(envelope.GetSeqId())),
			slog.Int("ret", int(response.GetRet())),
			slog.Int("money", int(response.GetMoney())),
			slog.Int("angry", int(response.GetAngry())),
			slog.Int("power", int(response.GetPower())),
			slog.Int("level_score", int(response.GetLevelScore())),
			slog.Int("num_of_shrew", int(response.GetNumOfShrew())),
			slog.Int("combo", int(response.GetCombo())),
			slog.Int("combo_id", int(response.GetComboId())),
		)
	}
}
