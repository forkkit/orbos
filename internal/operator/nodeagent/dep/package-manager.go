package dep

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/caos/orbos/mntr"
)

type Software struct {
	Package string
	Version string
}

func (s *Software) String() string {
	return fmt.Sprintf("%s=%s", s.Package, s.Version)
}

type Repository struct {
	Repository     string
	KeyURL         string
	KeyFingerprint string
}

type PackageManager struct {
	monitor   mntr.Monitor
	os        OperatingSystem
	installed map[string]string
	systemd   *SystemD
}

func (p *PackageManager) RefreshInstalled() error {
	var err error
	switch p.os.Packages {
	case DebianBased:
		err = p.debbasedInstalled()
	case REMBased:
		err = p.rembasedInstalled()
	}

	p.monitor.WithFields(map[string]interface{}{
		"packages": len(p.installed),
	}).Debug("Refreshed installed packages")

	return errors.Wrap(err, "refreshing installed packages failed")
}

func (p *PackageManager) Init() error {

	p.monitor.Debug("Initializing package manager")
	var err error
	switch p.os.Packages {
	case DebianBased:
		err = p.debSpecificUpdatePackages()
	case REMBased:
		err = p.remSpecificUpdatePackages()
	}

	if err != nil {
		return errors.Wrapf(err, "updating packages failed", p.os.Packages)
	}

	p.monitor.Debug("Package manager initialized")
	return nil
}

func NewPackageManager(monitor mntr.Monitor, os OperatingSystem, systemd *SystemD) *PackageManager {
	return &PackageManager{monitor, os, nil, systemd}
}

func (p *PackageManager) CurrentVersions(possiblePackages ...string) ([]*Software, error) {

	software := make([]*Software, 0)
	for _, pkg := range possiblePackages {
		if version, ok := p.installed[pkg]; ok {
			pkg := &Software{
				Package: pkg,
				Version: version,
			}
			software = append(software, pkg)
			p.monitor.WithFields(map[string]interface{}{
				"package": pkg.Package,
				"version": pkg.Version,
			}).Debug("Found filtered installed package")
		}
	}

	return software, nil
}

func (p *PackageManager) Install(installVersion *Software, more ...*Software) error {
	switch p.os.Packages {
	case DebianBased:
		return p.debbasedInstall(installVersion, more...)
	case REMBased:
		return p.rembasedInstall(installVersion, more...)
	}
	return errors.Errorf("Package manager %s is not implemented", p.os.Packages)
}

func (p *PackageManager) Add(repo *Repository) {
	var err error
	switch p.os.Packages {
	case DebianBased:
		err = p.debbasedAdd(repo)
	case REMBased:
		err = p.rembasedAdd(repo)
	default:
		err = errors.Errorf("Package manager %s is not implemented", p.os.Packages)
	}
	if err != nil {
		p.monitor.Info(errors.Wrap(err, "Adding repo failed").Error())
	}
}
