package runtime

import (
	"github.com/navire-dev/navire/dispatcher/internal/config"
	sharedcrypto "github.com/navire-dev/navire/shared/crypto"
)

func loadSecurity(cfg *config.Config) ([]byte, []byte, error) {
	dispatchPlanKey, err := sharedcrypto.ParseKey(cfg.DispatchPlanKey, "NAVIRE_DISPATCH_PLAN_KEY")
	if err != nil {
		return nil, nil, err
	}
	registrationSecret, err := sharedcrypto.ParseSecretKey(cfg.RegistrationSecret, "NAVIRE_DSPC_REGISTRATION_SECRET")
	if err != nil {
		return nil, nil, err
	}
	return dispatchPlanKey, registrationSecret, nil
}
