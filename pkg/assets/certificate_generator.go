package assets

import (
	"crypto/rand"
	"crypto/x509"
	"fmt"
	"math/big"
	"time"

	"github.com/go-logr/logr"
	"github.com/openshift/library-go/pkg/crypto"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/authentication/user"
)

const certificateLifetime = time.Duration(crypto.DefaultCertificateLifetimeInDays) * 24 * time.Hour
const GRPCSecretName = "thanos-grpc-secret"

// Taken from
// https://github.com/openshift/library-go/blob/08c2fd1b452520da35ad210930ea9d100545589a/pkg/operator/certrotation/signer.go#L68-L86
// without refresh time handling. We just take care of rotation if we reach 1/5 of the validity timespan before expiration.
func needsNewCert(notBefore, notAfter time.Time, now func() time.Time) bool {
	maxWait := notAfter.Sub(notBefore) / 5
	latestTime := notAfter.Add(-maxWait)
	return now().After(latestTime)
}

// Taken from
// https://github.com/openshift/cluster-monitoring-operator/blob/765d0b0369b176a5997d787b6710783437172879/pkg/manifests/tls.go#L113
func RotateGRPCSecret(s *v1.Secret, logger logr.Logger) (bool, error) {
	var (
		curCA, newCA              *crypto.CA
		curCABytes, crtPresent    = s.Data["ca.crt"]
		curCAKeyBytes, keyPresent = s.Data["ca.key"]
		rotate                    = !crtPresent || !keyPresent
	)

	if crtPresent && keyPresent {
		var err error
		curCA, err = crypto.GetCAFromBytes(curCABytes, curCAKeyBytes)
		if err != nil {
			logger.Info(fmt.Sprintf("generating a new CA due to error reading CA: %v", err))
			rotate = true
		} else if needsNewCert(curCA.Config.Certs[0].NotBefore, curCA.Config.Certs[0].NotAfter, time.Now) {
			logger.Info("generating new CA, because the current one is older than 1/5 of it validity timestamp")
			rotate = true
		}
	}

	if !rotate {
		return rotate, nil
	}

	if curCA == nil {
		newCAConfig, err := crypto.MakeSelfSignedCAConfig(
			fmt.Sprintf("%s@%d", "openshift-cluster-monitoring", time.Now().Unix()),
			crypto.DefaultCertificateLifetimeInDays,
		)
		if err != nil {
			return rotate, fmt.Errorf("error generating self signed CA: %w", err)
		}

		newCA = &crypto.CA{
			SerialGenerator: &crypto.RandomSerialGenerator{},
			Config:          newCAConfig,
		}
	} else {
		template := curCA.Config.Certs[0]
		now := time.Now()
		template.NotBefore = now.Add(-1 * time.Second)
		template.NotAfter = now.Add(certificateLifetime)
		template.SerialNumber = template.SerialNumber.Add(template.SerialNumber, big.NewInt(1))

		newCACert, err := createCertificate(template, template, template.PublicKey, curCA.Config.Key)
		if err != nil {
			return rotate, fmt.Errorf("error rotating CA: %w", err)
		}

		newCA = &crypto.CA{
			SerialGenerator: &crypto.RandomSerialGenerator{},
			Config: &crypto.TLSCertificateConfig{
				Certs: []*x509.Certificate{newCACert},
				Key:   curCA.Config.Key,
			},
		}
	}

	newCABytes, newCAKeyBytes, err := newCA.Config.GetPEMBytes()
	if err != nil {
		return rotate, fmt.Errorf("error getting PEM bytes from CA: %w", err)
	}

	s.Data["ca.crt"] = newCABytes
	s.Data["ca.key"] = newCAKeyBytes

	{
		cfg, err := newCA.MakeClientCertificateForDuration(
			&user.DefaultInfo{
				Name: "thanos-querier",
			},
			time.Duration(crypto.DefaultCertificateLifetimeInDays)*24*time.Hour,
		)
		if err != nil {
			return rotate, fmt.Errorf("error making client certificate: %w", err)
		}

		crt, key, err := cfg.GetPEMBytes()
		if err != nil {
			return rotate, fmt.Errorf("error getting PEM bytes for thanos querier client certificate: %w", err)
		}
		s.Data["thanos-querier-client.crt"] = crt
		s.Data["thanos-querier-client.key"] = key
	}

	{
		cfg, err := newCA.MakeServerCert(
			sets.NewString("prometheus-grpc"),
			crypto.DefaultCertificateLifetimeInDays,
		)
		if err != nil {
			return rotate, fmt.Errorf("error making server certificate: %w", err)
		}

		crt, key, err := cfg.GetPEMBytes()
		if err != nil {
			return rotate, fmt.Errorf("error getting PEM bytes for prometheus-k8s server certificate: %w", err)
		}
		s.Data["prometheus-server.crt"] = crt
		s.Data["prometheus-server.key"] = key
	}

	return rotate, nil
}

// createCertificate creates a new certificate and returns it in x509.Certificate form.
func createCertificate(template, parent *x509.Certificate, pub, priv interface{}) (*x509.Certificate, error) {
	rawCert, err := x509.CreateCertificate(rand.Reader, template, parent, pub, priv)
	if err != nil {
		return nil, fmt.Errorf("error creating certificate: %w", err)
	}
	parsedCerts, err := x509.ParseCertificates(rawCert)
	if err != nil {
		return nil, fmt.Errorf("error parsing certificate: %w", err)
	}
	return parsedCerts[0], nil
}
