package installer

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/syncloud/golib/config"
	"github.com/syncloud/golib/linux"
	"github.com/syncloud/golib/platform"
	"go.uber.org/zap"
)

const (
	App                  = "openvpn"
	prefixDelegationHook = "/etc/dhcp/dhclient-exit-hooks.d/openvpn"
)

type Variables struct {
	App              string
	AppDir           string
	DataDir          string
	CommonDir        string
	AppUrl           string
	AppDomain        string
	AuthUrl          string
	AuthSocket       string
	Domain           string
	DeviceDomain     string
	Secret           string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURI  string
}

type Installer struct {
	appDir             string
	dataDir            string
	commonDir          string
	configDir          string
	installFile        string
	newVersionFile     string
	currentVersionFile string
	platformClient     *platform.Client
	logger             *zap.Logger
}

func New(logger *zap.Logger) *Installer {
	appDir := path.Join("/snap", App, "current")
	dataDir := path.Join("/var/snap", App, "current")
	commonDir := path.Join("/var/snap", App, "common")

	return &Installer{
		appDir:             appDir,
		dataDir:            dataDir,
		commonDir:          commonDir,
		configDir:          path.Join(dataDir, "config"),
		installFile:        path.Join(dataDir, "installed"),
		newVersionFile:     path.Join(appDir, "version"),
		currentVersionFile: path.Join(dataDir, "version"),
		platformClient:     platform.New(),
		logger:             logger,
	}
}

func (i *Installer) Install() error {
	if err := linux.CreateUser(App); err != nil {
		return err
	}
	if err := i.createDirs(); err != nil {
		return err
	}
	if err := i.UpdateConfigs(); err != nil {
		return err
	}
	return i.StorageChange()
}

func (i *Installer) Configure() error {
	if i.IsInstalled() {
		if err := i.Upgrade(); err != nil {
			return err
		}
	} else {
		if err := i.Initialize(); err != nil {
			return err
		}
	}
	return i.UpdateVersion()
}

func (i *Installer) Initialize() error {
	if err := i.StorageChange(); err != nil {
		return err
	}
	return os.WriteFile(i.installFile, []byte("installed"), 0644)
}

func (i *Installer) Upgrade() error {
	return i.StorageChange()
}

func (i *Installer) IsInstalled() bool {
	_, err := os.Stat(i.installFile)
	return err == nil
}

func (i *Installer) PreRefresh() error {
	return nil
}

func (i *Installer) PostRefresh() error {
	if err := i.createDirs(); err != nil {
		return err
	}
	if err := i.UpdateConfigs(); err != nil {
		return err
	}
	return i.ClearVersion()
}

func (i *Installer) StorageChange() error {
	if _, err := i.platformClient.InitStorage(App, App); err != nil {
		return err
	}
	return i.fixPermissions()
}

func (i *Installer) AccessChange() error {
	return i.UpdateConfigs()
}

func (i *Installer) BackupPreStop() error {
	return i.PreRefresh()
}

func (i *Installer) RestorePreStart() error {
	return i.PostRefresh()
}

func (i *Installer) RestorePostStart() error {
	return i.Configure()
}

func (i *Installer) ClearVersion() error {
	return os.RemoveAll(i.currentVersionFile)
}

func (i *Installer) UpdateVersion() error {
	data, err := os.ReadFile(i.newVersionFile)
	if err != nil {
		return err
	}
	return os.WriteFile(i.currentVersionFile, data, 0644)
}

func (i *Installer) createDirs() error {
	return linux.CreateMissingDirs(
		i.dataDir,
		i.commonDir,
		i.configDir,
		path.Join(i.dataDir, "nginx"),
		path.Join(i.dataDir, "db"),
		path.Join(i.dataDir, "openvpn"),
		path.Join(i.dataDir, "pki"),
		path.Join(i.dataDir, "pki", "private"),
		path.Join(i.dataDir, "pki", "issued"),
		path.Join(i.dataDir, "pki", "reqs"),
	)
}

func (i *Installer) UpdateConfigs() error {
	if err := i.createDirs(); err != nil {
		return err
	}
	if err := MigrateLegacy(i.dataDir, i.logger); err != nil {
		return err
	}
	if err := i.linkPrefixDelegationHook(); err != nil {
		return err
	}

	appUrl, err := i.platformClient.GetAppUrl(App)
	if err != nil {
		return err
	}
	appDomain, err := i.platformClient.GetAppDomainName(App)
	if err != nil {
		return err
	}
	authUrl, err := i.platformClient.GetAppUrl("auth")
	if err != nil {
		return err
	}
	authSocket := strings.TrimSuffix(
		strings.TrimPrefix(i.platformClient.GetAuthLocalSocket(), "http://unix:"), ":")
	deviceDomain, err := i.platformClient.GetDeviceDomainName()
	if err != nil {
		return err
	}
	domain, found := strings.CutPrefix(appDomain, App+".")
	if !found {
		return fmt.Errorf("%s is not a prefix of %s", App, appDomain)
	}

	secret, err := getOrCreateSecret(path.Join(i.dataDir, ".secret"))
	if err != nil {
		return err
	}

	redirectPath := "/auth/callback"
	oidcSecret, err := i.platformClient.RegisterOIDCClient(
		App,
		[]string{redirectPath},
		true,
		"client_secret_basic",
	)
	if err != nil {
		return fmt.Errorf("register oidc client: %w", err)
	}

	variables := Variables{
		App:              App,
		AppDir:           i.appDir,
		DataDir:          i.dataDir,
		CommonDir:        i.commonDir,
		AppUrl:           appUrl,
		AppDomain:        appDomain,
		AuthUrl:          authUrl,
		AuthSocket:       authSocket,
		Domain:           domain,
		DeviceDomain:     deviceDomain,
		Secret:           secret,
		OIDCClientID:     App,
		OIDCClientSecret: oidcSecret,
		OIDCRedirectURI:  strings.TrimRight(appUrl, "/") + redirectPath,
	}

	if err := config.Generate(path.Join(i.appDir, "config"), i.configDir, variables); err != nil {
		return err
	}
	return i.fixPermissions()
}

func (i *Installer) linkPrefixDelegationHook() error {
	if err := linux.CreateMissingDirs(path.Dir(prefixDelegationHook)); err != nil {
		return err
	}
	if _, err := os.Lstat(prefixDelegationHook); err == nil {
		if err := os.Remove(prefixDelegationHook); err != nil {
			return err
		}
	}
	return os.Symlink(path.Join(i.appDir, "bin", "prefix_delegation.sh"), prefixDelegationHook)
}

func (i *Installer) fixPermissions() error {
	for _, dir := range []string{i.dataDir, i.commonDir} {
		if err := linux.Chown(dir, App); err != nil {
			return err
		}
	}
	return nil
}
