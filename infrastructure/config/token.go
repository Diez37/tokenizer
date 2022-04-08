package config

import (
	"github.com/diez37/go-packages/configurator"
	"time"
)

const (
	TokenRefreshFieldIp          = "ip"
	TokenRefreshFieldFingerprint = "fingerprint"
	TokenRefreshFieldUserAgent   = "user_agent"

	TokensAccessViolationActionDisableAll     = "disable_all"
	TokensAccessViolationActionDisableCurrent = "disable_current"
	TokensAccessViolationActionNone           = "none"

	TokensSecretFieldName                = "tokens.secret"
	TokensMaximumTokensFieldName         = "tokens.maximum"
	TokensDelayClearFieldName            = "tokens.delay.clear"
	TokensDelayBlockerFieldName          = "tokens.delay.blocker"
	TokensDelaySaverFieldName            = "tokens.delay.saver"
	TokensAccessLifetimeFieldName        = "tokens.access.lifetime"
	TokensRefreshLifetimeFieldName       = "tokens.refresh.lifetime"
	TokensCheckFieldsForRefreshFieldName = "tokens.refresh.check"
	TokensRefreshActionOnAccessViolation = "tokens.refresh.action.access_violation"

	TokensSecretDefault                = "fpbxsfhdYzd3U908O5hQ"
	TokensMaximumTokensDefault         = uint(5)
	TokensDelayClearDefault            = 10 * time.Second
	TokensDelayBlockerDefault          = 10 * time.Second
	TokensDelaySaverDefault            = 5 * time.Second
	TokensAccessLifetimeDefault        = 30 * time.Minute
	TokensRefreshLifetimeDefault       = time.Hour * 24 * 30 * 2
	TokensAccessViolationActionDefault = TokensAccessViolationActionDisableCurrent
)

var (
	TokensCheckFieldsForRefresh = []string{TokenRefreshFieldFingerprint, TokenRefreshFieldUserAgent}
)

type Token struct {
	Secret string

	MaximumTokens uint
	DelayClear    time.Duration
	DelayBlocker  time.Duration
	DelaySaver    time.Duration

	AccessLifetime  time.Duration
	RefreshLifetime time.Duration

	AccessViolation    string
	RefreshCheckFields []string
}

func NewToken() *Token {
	return &Token{}
}

func (config *Token) Configure(configurator configurator.Configurator) {
	configurator.SetDefault(TokensMaximumTokensFieldName, TokensMaximumTokensDefault)
	configurator.SetDefault(TokensDelayClearFieldName, TokensDelayClearDefault)
	configurator.SetDefault(TokensAccessLifetimeFieldName, TokensAccessLifetimeDefault)
	configurator.SetDefault(TokensRefreshLifetimeFieldName, TokensRefreshLifetimeDefault)

	if maximumTokens := configurator.GetUint(TokensMaximumTokensFieldName); config.MaximumTokens == 0 || config.MaximumTokens == TokensMaximumTokensDefault {
		config.MaximumTokens = maximumTokens
	}

	if delayClear := configurator.GetDuration(TokensDelayClearFieldName); config.DelayClear == 0 || config.DelayClear == TokensDelayClearDefault {
		config.DelayClear = delayClear
	}

	if lifetime := configurator.GetDuration(TokensAccessLifetimeFieldName); config.AccessLifetime == 0 || config.AccessLifetime == TokensAccessLifetimeDefault {
		config.AccessLifetime = lifetime
	}

	if lifetime := configurator.GetDuration(TokensRefreshLifetimeFieldName); config.RefreshLifetime == 0 || config.RefreshLifetime == TokensRefreshLifetimeDefault {
		config.RefreshLifetime = lifetime
	}
}
