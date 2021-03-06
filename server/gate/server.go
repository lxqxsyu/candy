package gate

import (
	"net"
	"sync"

	"github.com/juju/errors"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/dearcode/candy/server/meta"
	"github.com/dearcode/candy/server/util/log"
)

var (
	// ErrUndefineMethod 方法未定义.
	ErrUndefineMethod = errors.New("undefine method")
	// ErrInvalidContext 从context中解析客户端地址时出错.
	ErrInvalidContext = errors.New("invalid context")
	// ErrInvalidState 当前用户离线或未登录.
	ErrInvalidState = errors.New("invalid context")
)

// Gate recv client request.
type Gate struct {
	host     string
	master   *master
	store    *store
	sessions map[string]*session
	sync.RWMutex
}

// NewGate new gate server.
func NewGate(host, master, store string) *Gate {
	return &Gate{
		host:     host,
		master:   newMaster(master),
		store:    newStore(store),
		sessions: make(map[string]*session),
	}
}

// Start Gate service.
func (g *Gate) Start() error {
	log.Debugf("Gate Start...")
	serv := grpc.NewServer()
	meta.RegisterGateServer(serv, g)

	lis, err := net.Listen("tcp", g.host)
	if err != nil {
		return errors.Trace(err)
	}

	if err = g.master.start(); err != nil {
		return errors.Trace(err)
	}

	if err = g.store.start(); err != nil {
		return errors.Trace(err)
	}

	return serv.Serve(lis)
}

func (g *Gate) getSession(ctx context.Context) (*session, error) {
	log.Debug("Gate getSession")
	md, ok := metadata.FromContext(ctx)
	if !ok {
		return nil, errors.Trace(ErrInvalidContext)
	}

	addrs, ok := md["remote"]
	if !ok {
		return nil, errors.Trace(ErrInvalidContext)
	}

	g.RLock()
	s, ok := g.sessions[addrs[0]]
	g.RUnlock()

	if !ok {
		s = newSession(addrs[0])
		g.Lock()
		g.sessions[addrs[0]] = s
		g.Unlock()
	}

	return s, nil
}

func (g *Gate) getOnlineSession(ctx context.Context) (*session, error) {
	log.Debug("Gate getOnlineSession")
	s, err := g.getSession(ctx)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if !s.isOnline() {
		return nil, ErrInvalidState
	}

	return s, nil
}

// Register user, passwd.
func (g *Gate) Register(ctx context.Context, req *meta.GateRegisterRequest) (*meta.GateRegisterResponse, error) {
	log.Debug("Gate Register")
	_, err := g.getSession(ctx)
	if err != nil {
		log.Errorf("getSession error:%s", errors.ErrorStack(err))
		return nil, err
	}

	id, err := g.master.newID()
	if err != nil {
		return &meta.GateRegisterResponse{Header: &meta.ResponseHeader{Code: -1, Msg: err.Error()}}, nil
	}

	log.Debugf("Register user:%v password:%v", req.User, req.Password)

	if err = g.store.register(req.User, req.Password, id); err != nil {
		return &meta.GateRegisterResponse{Header: &meta.ResponseHeader{Code: -1, Msg: err.Error()}}, nil
	}

	return &meta.GateRegisterResponse{ID: id}, nil
}

// UpdateUserInfo nickname.
func (g *Gate) UpdateUserInfo(ctx context.Context, req *meta.GateUpdateUserInfoRequest) (*meta.GateUpdateUserInfoResponse, error) {
	log.Debug("Gate UpdateUserInfo")
	s, err := g.getSession(ctx)
	if err != nil {
		log.Errorf("getSession error:%s", errors.ErrorStack(err))
		return nil, err
	}

	if !s.isOnline() {
		err := errors.Errorf("current user is offline")
		return nil, err
	}

	log.Debugf("updateUserInfo user:%v niceName:%v", req.User, req.NickName)
	id, err := g.store.updateUserInfo(req.User, req.NickName, req.Avatar)
	if err != nil {
		return &meta.GateUpdateUserInfoResponse{Header: &meta.ResponseHeader{Code: -1, Msg: err.Error()}, ID: id}, nil
	}

	return &meta.GateUpdateUserInfoResponse{ID: id}, nil
}

// UpdateUserPassword update user password
func (g *Gate) UpdateUserPassword(ctx context.Context, req *meta.GateUpdateUserPasswordRequest) (*meta.GateUpdateUserPasswordResponse, error) {
	log.Debug("Gate UpdateUserPassword")
	return nil, ErrUndefineMethod
}

// GetUserInfo get user base info
func (g *Gate) GetUserInfo(ctx context.Context, req *meta.GateGetUserInfoRequest) (*meta.GateGetUserInfoResponse, error) {
	log.Debugf("Gate UserInfo")
	s, err := g.getSession(ctx)
	if err != nil {
		log.Errorf("getSession error:%s", errors.ErrorStack(err))
		return nil, err
	}

	if !s.isOnline() {
		err := errors.Errorf("current user is offline")
		return nil, err
	}

	log.Debugf("get UserInfo user:%v", req.User)
	id, name, nickName, avatar, err := g.store.getUserInfo(req.User)
	if err != nil {
		return &meta.GateGetUserInfoResponse{Header: &meta.ResponseHeader{Code: -1, Msg: err.Error()}}, nil
	}

	return &meta.GateGetUserInfoResponse{ID: id, User: name, NickName: nickName, Avatar: avatar}, nil
}

// Login user,passwd.
func (g *Gate) Login(ctx context.Context, req *meta.GateUserLoginRequest) (*meta.GateUserLoginResponse, error) {
	log.Debug("Gate Login")
	s, err := g.getSession(ctx)
	if err != nil {
		log.Errorf("getSession error:%s", errors.ErrorStack(err))
		return nil, err
	}

	log.Debugf("Login user:%v password:%v", req.User, req.Password)
	id, err := g.store.auth(req.User, req.Password)
	if err != nil {
		return &meta.GateUserLoginResponse{Header: &meta.ResponseHeader{Code: -1, Msg: err.Error()}}, nil
	}

	s.online(id)

	return &meta.GateUserLoginResponse{ID: id}, nil
}

// Logout nil.
func (g *Gate) Logout(ctx context.Context, req *meta.GateUserLogoutRequest) (*meta.GateUserLogoutResponse, error) {
	log.Debug("Gate Logout")
	return nil, ErrUndefineMethod
}

// UserMessage recv user message.
func (g *Gate) UserMessage(stream meta.Gate_UserMessageServer) error {
	log.Debugf("Gate UserMessage")
	for {
		break
	}

	return nil
}

// Heartbeat nil.
func (g *Gate) Heartbeat(ctx context.Context, req *meta.GateHeartbeatRequest) (*meta.GateHeartbeatResponse, error) {
	return nil, ErrUndefineMethod
}

// UploadImage image.
func (g *Gate) UploadImage(ctx context.Context, req *meta.GateUploadImageRequest) (*meta.GateUploadImageResponse, error) {
	return nil, ErrUndefineMethod
}

// DownloadImage ids.
func (g *Gate) DownloadImage(ctx context.Context, req *meta.GateDownloadImageRequest) (*meta.GateDownloadImageResponse, error) {
	return nil, ErrUndefineMethod
}

// Notice recv Notice server Message, and send Message to client.
func (g *Gate) Notice(ctx context.Context, req *meta.GateNoticeRequest) (*meta.GateNoticeResponse, error) {
	return nil, ErrUndefineMethod
}

// AddFriend 添加好友或确认接受添加.
func (g *Gate) AddFriend(ctx context.Context, req *meta.GateAddFriendRequest) (*meta.GateAddFriendResponse, error) {
	s, err := g.getOnlineSession(ctx)
	if err != nil {
		log.Errorf("getSession error:%s", errors.ErrorStack(err))
		return nil, err
	}

	ok, err := g.store.addFriend(s.getID(), req.UserID, req.Confirm)
	if err != nil {
		return &meta.GateAddFriendResponse{Header: &meta.ResponseHeader{Code: -1, Msg: err.Error()}}, nil
	}

	// 如果返回true，则可以直接聊天了，说明这是一个确认过的添加请求.
	return &meta.GateAddFriendResponse{Confirm: ok}, nil
}

// FindUser 添加好友前先查找.
func (g *Gate) FindUser(ctx context.Context, req *meta.GateFindUserRequest) (*meta.GateFindUserResponse, error) {
	_, err := g.getOnlineSession(ctx)
	if err != nil {
		log.Errorf("getSession error:%s", errors.ErrorStack(err))
		return nil, err
	}
	id, err := g.store.findUser(req.User)
	if err != nil {
		return &meta.GateFindUserResponse{Header: &meta.ResponseHeader{Code: -1, Msg: err.Error()}}, nil
	}
	return &meta.GateFindUserResponse{ID: id}, nil
}

// CreateGroup 用户创建一个聊天组.
func (g *Gate) CreateGroup(ctx context.Context, req *meta.GateCreateGroupRequest) (*meta.GateCreateGroupResponse, error) {
	s, err := g.getOnlineSession(ctx)
	if err != nil {
		log.Errorf("getSession error:%s", errors.ErrorStack(err))
		return nil, err
	}
	gid, err := g.master.newID()
	if err != nil {
		return &meta.GateCreateGroupResponse{Header: &meta.ResponseHeader{Code: -1, Msg: err.Error()}}, nil
	}

	if err = g.store.createGroup(s.getID(), gid); err != nil {
		return &meta.GateCreateGroupResponse{Header: &meta.ResponseHeader{Code: -1, Msg: err.Error()}}, nil
	}

	log.Debugf("user:%d, create group:%d", s.getID(), gid)
	return &meta.GateCreateGroupResponse{ID: gid}, nil
}
