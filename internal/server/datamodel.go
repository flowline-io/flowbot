package server

import (
	"encoding/json"
	"github.com/flowline-io/flowbot/internal/types"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// MsgGetOpts defines Get query parameters.
type MsgGetOpts struct {
	// Optional User ID to return result(s) for one user.
	User string `json:"user,omitempty"`
	// Optional topic name to return result(s) for one topic.
	Topic string `json:"topic,omitempty"`
	// Return results modified since this timespamp.
	IfModifiedSince *time.Time `json:"ims,omitempty"`
	// Load messages/ranges with IDs equal or greater than this (inclusive or closed)
	SinceId int `json:"since,omitempty"`
	// Load messages/ranges with IDs lower than this (exclusive or open)
	BeforeId int `json:"before,omitempty"`
	// Limit the number of messages loaded
	Limit int `json:"limit,omitempty"`
}

// MsgGetQuery is a topic metadata or data query.
type MsgGetQuery struct {
	What string `json:"what"`

	// Parameters of "desc" request: IfModifiedSince
	Desc *MsgGetOpts `json:"desc,omitempty"`
	// Parameters of "sub" request: User, Topic, IfModifiedSince, Limit.
	Sub *MsgGetOpts `json:"sub,omitempty"`
	// Parameters of "data" request: Since, Before, Limit.
	Data *MsgGetOpts `json:"data,omitempty"`
	// Parameters of "del" request: Since, Before, Limit.
	Del *MsgGetOpts `json:"del,omitempty"`
}

// MsgSetSub is a payload in set.sub request to update current subscription or invite another user, {sub.what} == "sub".
type MsgSetSub struct {
	// User affected by this request. Default (empty): current user
	User string `json:"user,omitempty"`

	// Access mode change, either Given or Want depending on context
	Mode string `json:"mode,omitempty"`
}

// MsgSetDesc is a C2S in set.what == "desc", acc, sub message.
type MsgSetDesc struct {
	DefaultAcs *MsgDefaultAcsMode `json:"defacs,omitempty"`  // default access mode
	Public     any                `json:"public,omitempty"`  // description of the user or topic
	Trusted    any                `json:"trusted,omitempty"` // trusted (system-provided) user or topic data
	Private    any                `json:"private,omitempty"` // per-subscription private data
}

// MsgCredClient is an account credential such as email or phone number.
type MsgCredClient struct {
	// Credential type, i.e. `email` or `tel`.
	Method string `json:"meth,omitempty"`
	// Value to verify, i.e. `user@example.com` or `+18003287448`
	Value string `json:"val,omitempty"`
	// Verification response
	Response string `json:"resp,omitempty"`
	// Request parameters, such as preferences. Passed to valiator without interpretation.
	Params map[string]any `json:"params,omitempty"`
}

// MsgSetQuery is an update to topic or user metadata: description, subscriptions, tags, credentials.
type MsgSetQuery struct {
	// Topic/user description, new object & new subscriptions only
	Desc *MsgSetDesc `json:"desc,omitempty"`
	// Subscription parameters
	Sub *MsgSetSub `json:"sub,omitempty"`
	// Indexable tags for user discovery
	Tags []string `json:"tags,omitempty"`
	// Update to account credentials.
	Cred *MsgCredClient `json:"cred,omitempty"`
}

// MsgDelRange is either an individual ID (HiId=0) or a randge of deleted IDs, low end inclusive (closed),
// high-end exclusive (open): [LowId .. HiId), e.g. 1..5 -> 1, 2, 3, 4.
type MsgDelRange struct {
	LowId int `json:"low,omitempty"`
	HiId  int `json:"hi,omitempty"`
}

/****************************************************************
 * Client to Server (C2S) messages.
 ****************************************************************/

// MsgClientHi is a handshake {hi} message.
type MsgClientHi struct {
	// Message Id
	Id string `json:"id,omitempty"`
	// User agent
	UserAgent string `json:"ua,omitempty"`
	// Protocol version, i.e. "0.13"
	Version string `json:"ver,omitempty"`
	// Client's unique device ID
	DeviceID string `json:"dev,omitempty"`
	// ISO 639-1 human language of the connected device
	Lang string `json:"lang,omitempty"`
	// Platform code: ios, android, web.
	Platform string `json:"platf,omitempty"`
	// Session is initially in non-iteractive, i.e. issued by a service. Presence notifications are delayed.
	Background bool `json:"bkg,omitempty"`
}

// MsgClientAcc is an {acc} message for creating or updating a user account.
type MsgClientAcc struct {
	// Message Id
	Id string `json:"id,omitempty"`
	// "newXYZ" to create a new user or UserId to update a user; default: current user.
	User string `json:"user,omitempty"`
	// Temporary authentication parameters for one-off actions, like password reset.
	TmpScheme string `json:"tmpscheme,omitempty"`
	TmpSecret []byte `json:"tmpsecret,omitempty"`
	// Account state: normal, suspended.
	State string `json:"status,omitempty"`
	// Authentication level of the user when UserID is set and not equal to the current user.
	// Either "", "auth" or "anon". Default: ""
	AuthLevel string `json:"authlevel,omitempty"`
	// The initial authentication scheme the account can use
	Scheme string `json:"scheme,omitempty"`
	// Shared secret
	Secret []byte `json:"secret,omitempty"`
	// Authenticate session with the newly created account
	Login bool `json:"login,omitempty"`
	// Indexable tags for user discovery
	Tags []string `json:"tags,omitempty"`
	// User initialization data when creating a new user, otherwise ignored
	Desc *MsgSetDesc `json:"desc,omitempty"`
	// Credentials to verify (email or phone or captcha)
	Cred []MsgCredClient `json:"cred,omitempty"`
}

// MsgClientLogin is a login {login} message.
type MsgClientLogin struct {
	// Message Id
	Id string `json:"id,omitempty"`
	// Authentication scheme
	Scheme string `json:"scheme,omitempty"`
	// Shared secret
	Secret []byte `json:"secret"`
	// Credntials being verified (email or phone or captcha etc.)
	Cred []MsgCredClient `json:"cred,omitempty"`
}

// MsgClientSub is a subscription request {sub} message.
type MsgClientSub struct {
	Id    string `json:"id,omitempty"`
	Topic string `json:"topic"`

	// Mirrors {set}.
	Set *MsgSetQuery `json:"set,omitempty"`

	// Mirrors {get}.
	Get *MsgGetQuery `json:"get,omitempty"`

	// Intra-cluster fields.

	// True if this subscription created a new topic.
	// In case of p2p topics, it's true if the other user's subscription was
	// created (as a part of new topic creation or just alone).
	Created bool `json:"-"`
	// True if this is a new subscription.
	Newsub bool `json:"-"`
}

// MsgDefaultAcsMode is a topic default access mode.
type MsgDefaultAcsMode struct {
	Auth string `json:"auth,omitempty"`
	Anon string `json:"anon,omitempty"`
}

// MsgClientLeave is an unsubscribe {leave} request message.
type MsgClientLeave struct {
	Id    string `json:"id,omitempty"`
	Topic string `json:"topic"`
	Unsub bool   `json:"unsub,omitempty"`
}

// MsgClientPub is client's request to publish data to topic subscribers {pub}.
type MsgClientPub struct {
	Id      string         `json:"id,omitempty"`
	Topic   string         `json:"topic"`
	NoEcho  bool           `json:"noecho,omitempty"`
	Head    map[string]any `json:"head,omitempty"`
	Content any            `json:"content"`
}

// MsgClientGet is a query of topic state {get}.
type MsgClientGet struct {
	Id    string `json:"id,omitempty"`
	Topic string `json:"topic"`
	MsgGetQuery
}

// MsgClientSet is an update of topic state {set}.
type MsgClientSet struct {
	Id    string `json:"id,omitempty"`
	Topic string `json:"topic"`
	MsgSetQuery
}

// MsgClientDel delete messages or topic {del}.
type MsgClientDel struct {
	Id    string `json:"id,omitempty"`
	Topic string `json:"topic,omitempty"`
	// What to delete:
	// * "msg" to delete messages (default)
	// * "topic" to delete the topic
	// * "sub" to delete a subscription to topic.
	// * "user" to delete or disable user.
	// * "cred" to delete credential (email or phone)
	What string `json:"what"`
	// Delete messages with these IDs (either one by one or a set of ranges)
	DelSeq []MsgDelRange `json:"delseq,omitempty"`
	// User ID of the user or subscription to delete
	User string `json:"user,omitempty"`
	// Credential to delete
	Cred *MsgCredClient `json:"cred,omitempty"`
	// Request to hard-delete objects (i.e. delete messages for all users), if such option is available.
	Hard bool `json:"hard,omitempty"`
}

// MsgClientNote is a client-generated notification for topic subscribers {note}.
type MsgClientNote struct {
	// There is no Id -- server will not akn {ping} packets, they are "fire and forget"
	Topic string `json:"topic"`
	// what is being reported: "recv" - message received, "read" - message read, "kp" - typing notification
	What string `json:"what"`
	// Server-issued message ID being reported
	SeqId int `json:"seq,omitempty"`
	// Client's count of unread messages to report back to the server. Used in push notifications on iOS.
	Unread int `json:"unread,omitempty"`
	// Call event.
	Event string `json:"event,omitempty"`
	// Arbitrary json payload (used in video calls).
	Payload json.RawMessage `json:"payload,omitempty"`
}

// MsgClientExtra is not a stand-alone message but extra data which augments the main payload.
type MsgClientExtra struct {
	// Array of out-of-band attachments which have to be exempted from GC.
	Attachments []string `json:"attachments,omitempty"`
	// Alternative user ID set by the root user (obo = On Behalf Of).
	AsUser string `json:"obo,omitempty"`
	// Altered authentication level set by the root user.
	AuthLevel string `json:"authlevel,omitempty"`
}

// ClientComMessage is a wrapper for client messages.
type ClientComMessage struct {
	Hi    *MsgClientHi    `json:"hi"`
	Acc   *MsgClientAcc   `json:"acc"`
	Login *MsgClientLogin `json:"login"`
	Sub   *MsgClientSub   `json:"sub"`
	Leave *MsgClientLeave `json:"leave"`
	Pub   *MsgClientPub   `json:"pub"`
	Get   *MsgClientGet   `json:"get"`
	Set   *MsgClientSet   `json:"set"`
	Del   *MsgClientDel   `json:"del"`
	Note  *MsgClientNote  `json:"note"`
	// Optional data.
	Extra *MsgClientExtra `json:"extra"`

	// Internal fields, routed only within the cluster.

	// Message ID denormalized
	Id string `json:"-"`
	// Un-routable (original) topic name denormalized from XXX.Topic.
	Original string `json:"-"`
	// Routable (expanded) topic name.
	RcptTo string `json:"-"`
	// Sender's UserId as string.
	AsUser string `json:"-"`
	// Sender's authentication level.
	AuthLvl int `json:"-"`
	// Denormalized 'what' field of meta messages (set, get, del).
	MetaWhat int `json:"-"`
	// Timestamp when this message was received by the server.
	Timestamp time.Time `json:"-"`

	// Originating session to send an aknowledgement to.
	sess *Session
	// The message is initialized (true) as opposite to being used as a wrapper for session.
	init bool
}

/****************************************************************
 * Server to client messages.
 ****************************************************************/

// MsgLastSeenInfo contains info on user's appearance online - when & user agent.
type MsgLastSeenInfo struct {
	// Timestamp of user's last appearance online.
	When *time.Time `json:"when,omitempty"`
	// User agent of the device when the user was last online.
	UserAgent string `json:"ua,omitempty"`
}

func (src *MsgLastSeenInfo) describe() string {
	return "'" + src.UserAgent + "' @ " + src.When.String()
}

// MsgCredServer is an account credential such as email or phone number.
type MsgCredServer struct {
	// Credential type, i.e. `email` or `tel`.
	Method string `json:"meth,omitempty"`
	// Credential value, i.e. `user@example.com` or `+18003287448`
	Value string `json:"val,omitempty"`
	// Indicates that the credential is validated.
	Done bool `json:"done,omitempty"`
}

// MsgAccessMode is a definition of access mode.
type MsgAccessMode struct {
	// Access mode requested by the user
	Want string `json:"want,omitempty"`
	// Access mode granted to the user by the admin
	Given string `json:"given,omitempty"`
	// Cumulative access mode want & given
	Mode string `json:"mode,omitempty"`
}

func (src *MsgAccessMode) describe() string {
	var s string
	if src.Want != "" {
		s = "w=" + src.Want
	}
	if src.Given != "" {
		s += " g=" + src.Given
	}
	if src.Mode != "" {
		s += " m=" + src.Mode
	}
	return strings.TrimSpace(s)
}

// MsgTopicDesc is a topic description, S2C in Meta message.
type MsgTopicDesc struct {
	CreatedAt *time.Time `json:"created,omitempty"`
	UpdatedAt *time.Time `json:"updated,omitempty"`
	// Timestamp of the last message
	TouchedAt *time.Time `json:"touched,omitempty"`

	// Account state, 'me' topic only.
	State string `json:"state,omitempty"`

	// If the group topic is online.
	Online bool `json:"online,omitempty"`

	// If the topic can be accessed as a channel
	IsChan bool `json:"chan,omitempty"`

	// P2P other user's last online timestamp & user agent
	LastSeen *MsgLastSeenInfo `json:"seen,omitempty"`

	DefaultAcs *MsgDefaultAcsMode `json:"defacs,omitempty"`
	// Actual access mode
	Acs *MsgAccessMode `json:"acs,omitempty"`
	// Max message ID
	SeqId     int `json:"seq,omitempty"`
	ReadSeqId int `json:"read,omitempty"`
	RecvSeqId int `json:"recv,omitempty"`
	// Id of the last delete operation as seen by the requesting user
	DelId   int `json:"clear,omitempty"`
	Public  any `json:"public,omitempty"`
	Trusted any `json:"trusted,omitempty"`
	// Per-subscription private data
	Private any `json:"private,omitempty"`
}

func (src *MsgTopicDesc) describe() string {
	var s string
	if src.State != "" {
		s = " state=" + src.State
	}
	s += " online=" + strconv.FormatBool(src.Online)
	if src.Acs != nil {
		s += " acs={" + src.Acs.describe() + "}"
	}
	if src.SeqId != 0 {
		s += " seq=" + strconv.Itoa(src.SeqId)
	}
	if src.ReadSeqId != 0 {
		s += " read=" + strconv.Itoa(src.ReadSeqId)
	}
	if src.RecvSeqId != 0 {
		s += " recv=" + strconv.Itoa(src.RecvSeqId)
	}
	if src.DelId != 0 {
		s += " clear=" + strconv.Itoa(src.DelId)
	}
	if src.Public != nil {
		s += " pub='...'"
	}
	if src.Trusted != nil {
		s += " trst='...'"
	}
	if src.Private != nil {
		s += " priv='...'"
	}
	return s
}

// MsgTopicSub is topic subscription details, sent in Meta message.
type MsgTopicSub struct {
	// Fields common to all subscriptions

	// Timestamp when the subscription was last updated
	UpdatedAt *time.Time `json:"updated,omitempty"`
	// Timestamp when the subscription was deleted
	DeletedAt *time.Time `json:"deleted,omitempty"`

	// If the subscriber/topic is online
	Online bool `json:"online,omitempty"`

	// Access mode. Topic admins receive the full info, non-admins receive just the cumulative mode
	// Acs.Mode = want & given. The field is not a pointer because at least one value is always assigned.
	Acs MsgAccessMode `json:"acs,omitempty"`
	// ID of the message reported by the given user as read
	ReadSeqId int `json:"read,omitempty"`
	// ID of the message reported by the given user as received
	RecvSeqId int `json:"recv,omitempty"`
	// Topic's public data
	Public any `json:"public,omitempty"`
	// Topic's trusted public data
	Trusted any `json:"trusted,omitempty"`
	// User's own private data per topic
	Private any `json:"private,omitempty"`

	// Response to non-'me' topic

	// Uid of the subscribed user
	User string `json:"user,omitempty"`

	// The following sections makes sense only in context of getting
	// user's own subscriptions ('me' topic response)

	// Topic name of this subscription
	Topic string `json:"topic,omitempty"`
	// Timestamp of the last message in the topic.
	TouchedAt *time.Time `json:"touched,omitempty"`
	// ID of the last {data} message in a topic
	SeqId int `json:"seq,omitempty"`
	// Id of the latest Delete operation
	DelId int `json:"clear,omitempty"`

	// P2P topics in 'me' {get subs} response:

	// Other user's last online timestamp & user agent
	LastSeen *MsgLastSeenInfo `json:"seen,omitempty"`
}

func (src *MsgTopicSub) describe() string {
	s := src.Topic + ":" + src.User + " online=" + strconv.FormatBool(src.Online) + " acs=" + src.Acs.describe()

	if src.SeqId != 0 {
		s += " seq=" + strconv.Itoa(src.SeqId)
	}
	if src.ReadSeqId != 0 {
		s += " read=" + strconv.Itoa(src.ReadSeqId)
	}
	if src.RecvSeqId != 0 {
		s += " recv=" + strconv.Itoa(src.RecvSeqId)
	}
	if src.DelId != 0 {
		s += " clear=" + strconv.Itoa(src.DelId)
	}
	if src.Public != nil {
		s += " pub='...'"
	}
	if src.Trusted != nil {
		s += " trst='...'"
	}
	if src.Private != nil {
		s += " priv='...'"
	}
	if src.LastSeen != nil {
		s += " seen={" + src.LastSeen.describe() + "}"
	}
	return s
}

// MsgDelValues describes request to delete messages.
type MsgDelValues struct {
	DelId  int           `json:"clear,omitempty"`
	DelSeq []MsgDelRange `json:"delseq,omitempty"`
}

// MsgServerCtrl is a server control message {ctrl}.
type MsgServerCtrl struct {
	Id     string `json:"id,omitempty"`
	Topic  string `json:"topic,omitempty"`
	Params any    `json:"params,omitempty"`

	Code      int       `json:"code"`
	Text      string    `json:"text,omitempty"`
	Timestamp time.Time `json:"ts"`
}

// Deep-shallow copy.
func (src *MsgServerCtrl) copy() *MsgServerCtrl {
	if src == nil {
		return nil
	}
	dst := *src
	return &dst
}

func (src *MsgServerCtrl) describe() string {
	return src.Topic + " id=" + src.Id + " code=" + strconv.Itoa(src.Code) + " txt=" + src.Text
}

// MsgServerData is a server {data} message.
type MsgServerData struct {
	Topic string `json:"topic"`
	// ID of the user who originated the message as {pub}, could be empty if sent by the system
	From      string         `json:"from,omitempty"`
	Timestamp time.Time      `json:"ts"`
	DeletedAt *time.Time     `json:"deleted,omitempty"`
	SeqId     int            `json:"seq"`
	Head      map[string]any `json:"head,omitempty"`
	Content   any            `json:"content"`
}

// Deep-shallow copy.
func (src *MsgServerData) copy() *MsgServerData {
	if src == nil {
		return nil
	}
	dst := *src
	return &dst
}

func (src *MsgServerData) describe() string {
	s := src.Topic + " from=" + src.From + " seq=" + strconv.Itoa(src.SeqId)
	if src.DeletedAt != nil {
		s += " deleted"
	} else {
		if src.Head != nil {
			s += " head=..."
		}
		s += " content='...'"
	}
	return s
}

// MsgServerPres is presence notification {pres} (authoritative update).
type MsgServerPres struct {
	Topic     string        `json:"topic"`
	Src       string        `json:"src,omitempty"`
	What      string        `json:"what"`
	UserAgent string        `json:"ua,omitempty"`
	SeqId     int           `json:"seq,omitempty"`
	DelId     int           `json:"clear,omitempty"`
	DelSeq    []MsgDelRange `json:"delseq,omitempty"`
	AcsTarget string        `json:"tgt,omitempty"`
	AcsActor  string        `json:"act,omitempty"`
	// Acs or a delta Acs. Need to marshal it to json under a name different than 'acs'
	// to allow different handling on the client
	Acs *MsgAccessMode `json:"dacs,omitempty"`

	// UNroutable params. All marked with `json:"-"` to exclude from json marshaling.
	// They are still serialized for intra-cluster communication.

	// Flag to break the reply loop
	WantReply bool `json:"-"`

	// Additional access mode filters when sending to topic's online members. Both filter conditions must be true.
	// send only to those who have this access mode.
	FilterIn int `json:"-"`
	// skip those who have this access mode.
	FilterOut int `json:"-"`

	// When sending to 'me', skip sessions subscribed to this topic.
	SkipTopic string `json:"-"`

	// Send to sessions of a single user only.
	SingleUser string `json:"-"`

	// Exclude sessions of a single user.
	ExcludeUser string `json:"-"`
}

// Deep-shallow copy.
func (src *MsgServerPres) copy() *MsgServerPres {
	if src == nil {
		return nil
	}
	dst := *src
	return &dst
}

func (src *MsgServerPres) describe() string {
	s := src.Topic
	if src.Src != "" {
		s += " src=" + src.Src
	}
	if src.What != "" {
		s += " what=" + src.What
	}
	if src.UserAgent != "" {
		s += " ua=" + src.UserAgent
	}
	if src.SeqId != 0 {
		s += " seq=" + strconv.Itoa(src.SeqId)
	}
	if src.DelId != 0 {
		s += " clear=" + strconv.Itoa(src.DelId)
	}
	if src.DelSeq != nil {
		s += " delseq"
	}
	if src.AcsTarget != "" {
		s += " tgt=" + src.AcsTarget
	}
	if src.AcsActor != "" {
		s += " actor=" + src.AcsActor
	}
	if src.Acs != nil {
		s += " dacs=" + src.Acs.describe()
	}

	return s
}

// MsgServerMeta is a topic metadata {meta} update.
type MsgServerMeta struct {
	Id    string `json:"id,omitempty"`
	Topic string `json:"topic"`

	Timestamp *time.Time `json:"ts,omitempty"`

	// Topic description
	Desc *MsgTopicDesc `json:"desc,omitempty"`
	// Subscriptions as an array of objects
	Sub []MsgTopicSub `json:"sub,omitempty"`
	// Delete ID and the ranges of IDs of deleted messages
	Del *MsgDelValues `json:"del,omitempty"`
	// User discovery tags
	Tags []string `json:"tags,omitempty"`
	// Account credentials, 'me' only.
	Cred []*MsgCredServer `json:"cred,omitempty"`
}

// Deep-shallow copy of meta message. Deep copy of Id and Topic fields, shallow copy of payload.
func (src *MsgServerMeta) copy() *MsgServerMeta {
	if src == nil {
		return nil
	}
	dst := *src
	return &dst
}

func (src *MsgServerMeta) describe() string {
	s := src.Topic + " id=" + src.Id

	if src.Desc != nil {
		s += " desc={" + src.Desc.describe() + "}"
	}
	if src.Sub != nil {
		var x []string
		for _, sub := range src.Sub {
			x = append(x, sub.describe())
		}
		s += " sub=[{" + strings.Join(x, "},{") + "}]"
	}
	if src.Del != nil {
		x, _ := json.Marshal(src.Del)
		s += " del={" + string(x) + "}"
	}
	if src.Tags != nil {
		s += " tags=[" + strings.Join(src.Tags, ",") + "]"
	}
	if src.Cred != nil {
		x, _ := json.Marshal(src.Cred)
		s += " cred=[" + string(x) + "]"
	}
	return s
}

// MsgServerInfo is the server-side copy of MsgClientNote with From and optionally Src added (non-authoritative).
type MsgServerInfo struct {
	// Topic to send event to.
	Topic string `json:"topic"`
	// Topic where the even has occurred (set only when Topic='me').
	Src string `json:"src,omitempty"`
	// ID of the user who originated the message.
	From string `json:"from,omitempty"`
	// The event being reported: "rcpt" - message received, "read" - message read, "kp" - typing notification, "call" - video call.
	What string `json:"what"`
	// Server-issued message ID being reported.
	SeqId int `json:"seq,omitempty"`
	// Call event.
	Event string `json:"event,omitempty"`
	// Arbitrary json payload (used by video calls).
	Payload json.RawMessage `json:"payload,omitempty"`

	// UNroutable params. All marked with `json:"-"` to exclude from json marshaling.
	// They are still serialized for intra-cluster communication.

	// When sending to 'me', skip sessions subscribed to this topic.
	SkipTopic string `json:"-"`
}

// Deep copy.
func (src *MsgServerInfo) copy() *MsgServerInfo {
	if src == nil {
		return nil
	}
	dst := *src
	return &dst
}

// Basic description.
func (src *MsgServerInfo) describe() string {
	s := src.Topic
	if src.Src != "" {
		s += " src=" + src.Src
	}
	s += " what=" + src.What + " from=" + src.From
	if src.SeqId > 0 {
		s += " seq=" + strconv.Itoa(src.SeqId)
	}
	if len(src.Payload) > 0 {
		s += " payload=<..." + strconv.Itoa(len(src.Payload)) + " bytes ...>"
	}
	return s
}

// ServerComMessage is a wrapper for server-side messages.
type ServerComMessage struct {
	Ctrl *MsgServerCtrl `json:"ctrl,omitempty"`
	Data *MsgServerData `json:"data,omitempty"`
	Meta *MsgServerMeta `json:"meta,omitempty"`
	Pres *MsgServerPres `json:"pres,omitempty"`
	Info *MsgServerInfo `json:"info,omitempty"`

	// Internal fields.

	// MsgServerData has no Id field, copying it here for use in {ctrl} aknowledgements
	Id string `json:"-"`
	// Routable (expanded) name of the topic.
	RcptTo string `json:"-"`
	// User ID of the sender of the original message.
	AsUser string `json:"-"`
	// Timestamp for consistency of timestamps in {ctrl} messages
	// (corresponds to originating client message receipt timestamp).
	Timestamp time.Time `json:"-"`
	// Originating session to send an aknowledgement to. Could be nil.
	sess *Session
	// Session ID to skip when sendng packet to sessions. Used to skip sending to original session.
	// Could be either empty.
	SkipSid string `json:"-"`
	// User id affected by this message.
	uid types.Uid
}

// Deep-shallow copy of ServerComMessage. Deep copy of service fields,
// shallow copy of session and payload.
func (src *ServerComMessage) copy() *ServerComMessage {
	if src == nil {
		return nil
	}
	dst := &ServerComMessage{
		Id:        src.Id,
		RcptTo:    src.RcptTo,
		AsUser:    src.AsUser,
		Timestamp: src.Timestamp,
		sess:      src.sess,
		SkipSid:   src.SkipSid,
		uid:       src.uid,
	}

	dst.Ctrl = src.Ctrl.copy()
	dst.Data = src.Data.copy()
	dst.Meta = src.Meta.copy()
	dst.Pres = src.Pres.copy()
	dst.Info = src.Info.copy()

	return dst
}

func (src *ServerComMessage) describe() string {
	if src == nil {
		return "-"
	}

	switch {
	case src.Ctrl != nil:
		return "{ctrl " + src.Ctrl.describe() + "}"
	case src.Data != nil:
		return "{data " + src.Data.describe() + "}"
	case src.Meta != nil:
		return "{meta " + src.Meta.describe() + "}"
	case src.Pres != nil:
		return "{pres " + src.Pres.describe() + "}"
	case src.Info != nil:
		return "{info " + src.Info.describe() + "}"
	default:
		return "{nil}"
	}
}

// Generators of server-side error messages {ctrl}.

// NoErrEvicted indicates that the user was disconnected from topic for no fault of the user (205).
func NoErrEvicted(id, topic string, ts time.Time) *ServerComMessage {
	return &ServerComMessage{
		Ctrl: &MsgServerCtrl{
			Id:        id,
			Code:      http.StatusResetContent, // 205
			Text:      "evicted",
			Topic:     topic,
			Timestamp: ts,
		}, Id: id,
	}
}

// NoErrShutdown means user was disconnected from topic because system shutdown is in progress (205).
func NoErrShutdown(ts time.Time) *ServerComMessage {
	return &ServerComMessage{
		Ctrl: &MsgServerCtrl{
			Code:      http.StatusResetContent, // 205
			Text:      "server shutdown",
			Timestamp: ts,
		},
	}
}

// 4xx Errors

// ErrMalformed request malformed (400).
func ErrMalformed(id, topic string, ts time.Time) *types.ServerComMessage {
	return ErrMalformedExplicitTs(id, topic, ts, ts)
}

// ErrMalformedExplicitTs request malformed with explicit server and incoming request timestamps (400).
func ErrMalformedExplicitTs(_, _ string, _, _ time.Time) *types.ServerComMessage {
	return &types.ServerComMessage{
		Code:    http.StatusBadRequest,
		Message: "ErrMalformedExplicitTs",
		Data:    nil,
	}
}

// ErrLocked operation rejected because the topic is being deleted (503).
func ErrLocked(id, topic string, ts time.Time) *types.ServerComMessage {
	return ErrLockedExplicitTs(id, topic, ts, ts)
}

// ErrLockedExplicitTs operation rejected because the topic is being deleted
// with explicit server and incoming request timestamps (503).
func ErrLockedExplicitTs(_, _ string, _, _ time.Time) *types.ServerComMessage {
	return &types.ServerComMessage{
		Code:    http.StatusServiceUnavailable,
		Message: "ErrLockedExplicitTs",
		Data:    nil,
	}
}
