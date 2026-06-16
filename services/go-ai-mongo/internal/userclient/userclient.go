// Package userclient is a gRPC client to the C# Supabase engine (orsa.user.v1.
// UserService). It uses dynamic protobuf descriptors so no generated stubs /
// protoc are required, matching the descriptor approach used elsewhere in this
// service. Service-to-service traffic is gRPC; the browser-facing API is REST.
package userclient

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type descriptors struct {
	userIDRequest        protoreflect.MessageDescriptor
	userSettings         protoreflect.MessageDescriptor
	updateSettingsReq    protoreflect.MessageDescriptor
	profileResponse      protoreflect.MessageDescriptor
	updateProfileReq     protoreflect.MessageDescriptor
	legalRequest         protoreflect.MessageDescriptor
	writeAck             protoreflect.MessageDescriptor
	attachmentUsage      protoreflect.MessageDescriptor
	consumeAttachmentReq protoreflect.MessageDescriptor
}

// Client talks to the C# UserService.
type Client struct {
	conn *grpc.ClientConn
	d    descriptors
}

// Settings is the decoded user settings view.
type Settings struct {
	MemoryExtractionEnabled bool
	RemindersEnabled        bool
	AttachmentCountToday    int
	AttachmentLimit         int
}

// AttachmentUsage is the decoded per-user/per-day attachment quota state.
type AttachmentUsage struct {
	UsedToday  int
	Limit      int
	Allowed    bool
	ResetAtISO string
}

// Profile is the decoded user profile view.
type Profile struct {
	DisplayName      string
	Country          string
	Region           string
	City             string
	PersonaJSON      string
	PersonaUpdatedAt string
	PersonaSummary   string
	WorkflowBoundary string
	ConsentStatus    string
	BoundaryPrompt   string
}

// New dials the C# gRPC endpoint (lazy connect) and builds message descriptors.
func New(target string) (*Client, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	d, err := buildDescriptors()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &Client{conn: conn, d: d}, nil
}

func (c *Client) Close() error { return c.conn.Close() }

func (c *Client) invoke(ctx context.Context, method string, req, resp *dynamicpb.Message) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return c.conn.Invoke(ctx, method, req, resp)
}

func (c *Client) GetSettings(ctx context.Context, userID string) (Settings, error) {
	req := dynamicpb.NewMessage(c.d.userIDRequest)
	setString(req, "api_version", "v1")
	setString(req, "user_id", userID)
	resp := dynamicpb.NewMessage(c.d.userSettings)
	if err := c.invoke(ctx, "/orsa.user.v1.UserService/GetSettings", req, resp); err != nil {
		return Settings{}, err
	}
	return decodeSettings(resp), nil
}

// UpdateSettings sends only the fields that are non-nil (proto3 presence).
func (c *Client) UpdateSettings(ctx context.Context, userID string, memory, reminders *bool) (Settings, error) {
	req := dynamicpb.NewMessage(c.d.updateSettingsReq)
	setString(req, "api_version", "v1")
	setString(req, "user_id", userID)
	if memory != nil {
		req.Set(req.Descriptor().Fields().ByName("memory_extraction_enabled"), protoreflect.ValueOfBool(*memory))
	}
	if reminders != nil {
		req.Set(req.Descriptor().Fields().ByName("reminders_enabled"), protoreflect.ValueOfBool(*reminders))
	}
	resp := dynamicpb.NewMessage(c.d.userSettings)
	if err := c.invoke(ctx, "/orsa.user.v1.UserService/UpdateSettings", req, resp); err != nil {
		return Settings{}, err
	}
	return decodeSettings(resp), nil
}

func (c *Client) GetProfile(ctx context.Context, userID string) (Profile, error) {
	req := dynamicpb.NewMessage(c.d.userIDRequest)
	setString(req, "api_version", "v1")
	setString(req, "user_id", userID)
	resp := dynamicpb.NewMessage(c.d.profileResponse)
	if err := c.invoke(ctx, "/orsa.user.v1.UserService/GetProfile", req, resp); err != nil {
		return Profile{}, err
	}
	return Profile{
		DisplayName:      getString(resp, "display_name"),
		Country:          getString(resp, "country"),
		Region:           getString(resp, "region"),
		City:             getString(resp, "city"),
		PersonaJSON:      getString(resp, "persona_json"),
		PersonaUpdatedAt: getString(resp, "persona_updated_at"),
		PersonaSummary:   getString(resp, "persona_summary"),
		WorkflowBoundary: getString(resp, "workflow_boundary"),
		ConsentStatus:    getString(resp, "consent_status"),
		BoundaryPrompt:   getString(resp, "boundary_prompt"),
	}, nil
}

func (c *Client) UpdateProfile(ctx context.Context, userID string, memory *bool, summary, boundary *string) (Profile, error) {
	req := dynamicpb.NewMessage(c.d.updateProfileReq)
	setString(req, "api_version", "v1")
	setString(req, "user_id", userID)
	if memory != nil {
		req.Set(req.Descriptor().Fields().ByName("memory_extraction_enabled"), protoreflect.ValueOfBool(*memory))
	}
	if summary != nil {
		req.Set(req.Descriptor().Fields().ByName("persona_summary"), protoreflect.ValueOfString(*summary))
	}
	if boundary != nil {
		req.Set(req.Descriptor().Fields().ByName("workflow_boundary"), protoreflect.ValueOfString(*boundary))
	}
	resp := dynamicpb.NewMessage(c.d.profileResponse)
	if err := c.invoke(ctx, "/orsa.user.v1.UserService/UpdateProfile", req, resp); err != nil {
		return Profile{}, err
	}
	return Profile{
		DisplayName:      getString(resp, "display_name"),
		Country:          getString(resp, "country"),
		Region:           getString(resp, "region"),
		City:             getString(resp, "city"),
		PersonaJSON:      getString(resp, "persona_json"),
		PersonaUpdatedAt: getString(resp, "persona_updated_at"),
		PersonaSummary:   getString(resp, "persona_summary"),
		WorkflowBoundary: getString(resp, "workflow_boundary"),
		ConsentStatus:    getString(resp, "consent_status"),
		BoundaryPrompt:   getString(resp, "boundary_prompt"),
	}, nil
}

func (c *Client) RecordLegalAcceptance(ctx context.Context, userID, terms, privacy, consent, acceptedAtISO string) error {
	req := dynamicpb.NewMessage(c.d.legalRequest)
	setString(req, "api_version", "v1")
	setString(req, "user_id", userID)
	setString(req, "terms_version", terms)
	setString(req, "privacy_version", privacy)
	setString(req, "consent_version", consent)
	setString(req, "accepted_at_iso", acceptedAtISO)
	resp := dynamicpb.NewMessage(c.d.writeAck)
	return c.invoke(ctx, "/orsa.user.v1.UserService/RecordLegalAcceptance", req, resp)
}

func (c *Client) GetAttachmentUsage(ctx context.Context, userID string) (AttachmentUsage, error) {
	req := dynamicpb.NewMessage(c.d.userIDRequest)
	setString(req, "api_version", "v1")
	setString(req, "user_id", userID)
	resp := dynamicpb.NewMessage(c.d.attachmentUsage)
	if err := c.invoke(ctx, "/orsa.user.v1.UserService/GetAttachmentUsage", req, resp); err != nil {
		return AttachmentUsage{}, err
	}
	return decodeUsage(resp), nil
}

func (c *Client) ConsumeAttachment(ctx context.Context, userID string, count, limit int) (AttachmentUsage, error) {
	req := dynamicpb.NewMessage(c.d.consumeAttachmentReq)
	setString(req, "api_version", "v1")
	setString(req, "user_id", userID)
	setInt32(req, "count", int32(count))
	setInt32(req, "limit", int32(limit))
	resp := dynamicpb.NewMessage(c.d.attachmentUsage)
	if err := c.invoke(ctx, "/orsa.user.v1.UserService/ConsumeAttachment", req, resp); err != nil {
		return AttachmentUsage{}, err
	}
	return decodeUsage(resp), nil
}

// DeleteUser permanently removes the user's Postgres-side records (settings,
// legal acceptances, persona audits, email verifications, attachment usage, and
// the user row). Chat threads are removed separately by the gateway.
func (c *Client) DeleteUser(ctx context.Context, userID string) error {
	req := dynamicpb.NewMessage(c.d.userIDRequest)
	setString(req, "api_version", "v1")
	setString(req, "user_id", userID)
	resp := dynamicpb.NewMessage(c.d.writeAck)
	return c.invoke(ctx, "/orsa.user.v1.UserService/DeleteUser", req, resp)
}

func decodeUsage(resp *dynamicpb.Message) AttachmentUsage {
	return AttachmentUsage{
		UsedToday:  int(getInt32(resp, "used_today")),
		Limit:      int(getInt32(resp, "limit")),
		Allowed:    getBool(resp, "allowed"),
		ResetAtISO: getString(resp, "reset_at_iso"),
	}
}

func decodeSettings(resp *dynamicpb.Message) Settings {
	return Settings{
		MemoryExtractionEnabled: getBool(resp, "memory_extraction_enabled"),
		RemindersEnabled:        getBool(resp, "reminders_enabled"),
		AttachmentCountToday:    int(getInt32(resp, "attachment_count_today")),
		AttachmentLimit:         int(getInt32(resp, "attachment_limit")),
	}
}

// ---- dynamic descriptors mirroring user_service.proto ----

func buildDescriptors() (descriptors, error) {
	file := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("user_service.proto"),
		Package: proto.String("orsa.user.v1"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			msg("UserIdRequest", strField("api_version", 1), strField("user_id", 2)),
			msg("UserSettings",
				strField("api_version", 1), strField("user_id", 2),
				boolField("memory_extraction_enabled", 3), boolField("reminders_enabled", 4),
				int32Field("attachment_count_today", 5), int32Field("attachment_limit", 6)),
			{
				Name: proto.String("UpdateSettingsRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					strField("api_version", 1), strField("user_id", 2),
					optionalBoolField("memory_extraction_enabled", 3, 0),
					optionalBoolField("reminders_enabled", 4, 1),
				},
				OneofDecl: []*descriptorpb.OneofDescriptorProto{
					{Name: proto.String("_memory_extraction_enabled")},
					{Name: proto.String("_reminders_enabled")},
				},
			},
			msg("ProfileResponse",
				strField("api_version", 1), strField("user_id", 2), strField("display_name", 3),
				strField("country", 4), strField("region", 5), strField("city", 6),
				strField("persona_json", 7), strField("persona_updated_at", 8),
				strField("persona_summary", 9), strField("workflow_boundary", 10),
				strField("consent_status", 11), strField("boundary_prompt", 12)),
			{
				Name: proto.String("UpdateProfileRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					strField("api_version", 1), strField("user_id", 2),
					optionalBoolField("memory_extraction_enabled", 3, 0),
					optionalStringField("persona_summary", 4, 1),
					optionalStringField("workflow_boundary", 5, 2),
				},
				OneofDecl: []*descriptorpb.OneofDescriptorProto{
					{Name: proto.String("_memory_extraction_enabled")},
					{Name: proto.String("_persona_summary")},
					{Name: proto.String("_workflow_boundary")},
				},
			},
			msg("LegalAcceptanceRequest",
				strField("api_version", 1), strField("user_id", 2), strField("terms_version", 3),
				strField("privacy_version", 4), strField("consent_version", 5), strField("accepted_at_iso", 6)),
			msg("WriteAck", strField("api_version", 1), boolField("ok", 2), strField("id", 3)),
			msg("AttachmentUsage",
				strField("api_version", 1), strField("user_id", 2),
				int32Field("used_today", 3), int32Field("limit", 4),
				boolField("allowed", 5), strField("reset_at_iso", 6)),
			msg("ConsumeAttachmentRequest",
				strField("api_version", 1), strField("user_id", 2),
				int32Field("count", 3), int32Field("limit", 4)),
		},
	}
	fd, err := protodesc.NewFile(file, nil)
	if err != nil {
		return descriptors{}, err
	}
	m := fd.Messages()
	d := descriptors{
		userIDRequest:        m.ByName("UserIdRequest"),
		userSettings:         m.ByName("UserSettings"),
		updateSettingsReq:    m.ByName("UpdateSettingsRequest"),
		profileResponse:      m.ByName("ProfileResponse"),
		updateProfileReq:     m.ByName("UpdateProfileRequest"),
		legalRequest:         m.ByName("LegalAcceptanceRequest"),
		writeAck:             m.ByName("WriteAck"),
		attachmentUsage:      m.ByName("AttachmentUsage"),
		consumeAttachmentReq: m.ByName("ConsumeAttachmentRequest"),
	}
	if d.userSettings == nil || d.updateSettingsReq == nil || d.updateProfileReq == nil ||
		d.attachmentUsage == nil || d.consumeAttachmentReq == nil {
		return descriptors{}, fmt.Errorf("failed to resolve user_service descriptors")
	}
	return d, nil
}

func msg(name string, fields ...*descriptorpb.FieldDescriptorProto) *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{Name: proto.String(name), Field: fields}
}

func strField(name string, number int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name: proto.String(name), Number: proto.Int32(number),
		Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:  descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
	}
}

func boolField(name string, number int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name: proto.String(name), Number: proto.Int32(number),
		Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:  descriptorpb.FieldDescriptorProto_TYPE_BOOL.Enum(),
	}
}

func int32Field(name string, number int32) *descriptorpb.FieldDescriptorProto {
	return &descriptorpb.FieldDescriptorProto{
		Name: proto.String(name), Number: proto.Int32(number),
		Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
		Type:  descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
	}
}

func optionalBoolField(name string, number, oneofIndex int32) *descriptorpb.FieldDescriptorProto {
	f := boolField(name, number)
	f.OneofIndex = proto.Int32(oneofIndex)
	f.Proto3Optional = proto.Bool(true)
	return f
}

func optionalStringField(name string, number, oneofIndex int32) *descriptorpb.FieldDescriptorProto {
	f := strField(name, number)
	f.OneofIndex = proto.Int32(oneofIndex)
	f.Proto3Optional = proto.Bool(true)
	return f
}

func setString(m *dynamicpb.Message, field protoreflect.Name, value string) {
	m.Set(m.Descriptor().Fields().ByName(field), protoreflect.ValueOfString(value))
}

func setInt32(m *dynamicpb.Message, field protoreflect.Name, value int32) {
	m.Set(m.Descriptor().Fields().ByName(field), protoreflect.ValueOfInt32(value))
}

func getString(m *dynamicpb.Message, field protoreflect.Name) string {
	f := m.Descriptor().Fields().ByName(field)
	if f == nil {
		return ""
	}
	return m.Get(f).String()
}

func getBool(m *dynamicpb.Message, field protoreflect.Name) bool {
	f := m.Descriptor().Fields().ByName(field)
	if f == nil {
		return false
	}
	return m.Get(f).Bool()
}

func getInt32(m *dynamicpb.Message, field protoreflect.Name) int32 {
	f := m.Descriptor().Fields().ByName(field)
	if f == nil {
		return 0
	}
	return int32(m.Get(f).Int())
}
