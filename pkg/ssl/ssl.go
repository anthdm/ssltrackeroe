package ssl

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/anthdm/ssltracker/data"
	"github.com/anthdm/ssltracker/logger"
)

func PollDomain(ctx context.Context, domain string) (*data.DomainTrackingInfo, error) {
	var (
		start    = time.Now()
		resultch = make(chan data.DomainTrackingInfo)
		config   = &tls.Config{}
	)
	go func() {
		conn, err := tls.Dial("tcp", fmt.Sprintf("%s:443", domain), config)
		if err != nil {
			info := data.DomainTrackingInfo{
				LastPollAt: time.Now(),
				Error:      err.Error(),
				Latency:    int(time.Since(start).Milliseconds()),
			}
			if IsVerificationError(err) {
				info.Status = data.StatusInvalid
				resultch <- info
			}
			if IsConnectionRefused(err) {
				info.Status = data.StatusOffline
				resultch <- info
			}
			return
		}
		defer conn.Close()
		var (
			state     = conn.ConnectionState()
			cert      = state.PeerCertificates[0]
			keyUsages = make([]string, len(cert.ExtKeyUsage))
			i         = 0
		)
		for _, usage := range cert.ExtKeyUsage {
			keyUsages[i] = extKeyUsageToString(usage)
			i++
		}
		host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
		if err != nil {
			logger.Log("error", err)
		}
		resultch <- data.DomainTrackingInfo{
			ServerIP:      host,
			PublicKeyAlgo: cert.PublicKeyAlgorithm.String(),
			SignatureAlgo: cert.SignatureAlgorithm.String(),
			KeyUsage:      keyUsageToString(cert.KeyUsage),
			ExtKeyUsages:  keyUsages,
			PublicKey:     publicKeyFromCert(cert),
			EncodedPEM:    encodedPEMFromCert(cert),
			Signature:     sha1Hex(cert.Signature),
			Expires:       cert.NotAfter,
			DNSNames:      strings.Join(cert.DNSNames, ", "),
			Issuer:        cert.Issuer.Organization[0],
			LastPollAt:    time.Now(),
			Latency:       int(time.Since(start).Milliseconds()),
			Status:        getStatus(cert.NotAfter),
		}
	}()

	select {
	case <-ctx.Done():
		return &data.DomainTrackingInfo{
			Error:      ctx.Err().Error(),
			LastPollAt: time.Now(),
			Status:     data.StatusUnresponsive,
		}, nil
	case result := <-resultch:
		return &result, nil
	}
}

func extKeyUsageToString(usage x509.ExtKeyUsage) string {
	switch usage {
	case x509.ExtKeyUsageAny:
		return "any"
	case x509.ExtKeyUsageServerAuth:
		return "server auth"
	case x509.ExtKeyUsageClientAuth:
		return "client auth"
	case x509.ExtKeyUsageCodeSigning:
		return "code signing"
	case x509.ExtKeyUsageEmailProtection:
		return "email protection"
	case x509.ExtKeyUsageIPSECEndSystem:
		return "IPS SEC system"
	case x509.ExtKeyUsageIPSECTunnel:
		return "IPS SEC tunnel"
	case x509.ExtKeyUsageIPSECUser:
		return "IPS SEC user"
	case x509.ExtKeyUsageTimeStamping:
		return "time stamping"
	case x509.ExtKeyUsageOCSPSigning:
		return "OCSP signing"
	case x509.ExtKeyUsageMicrosoftServerGatedCrypto:
		return "Microsoft server gated crypto"
	case x509.ExtKeyUsageNetscapeServerGatedCrypto:
		return "Netscape server gated crypto"
	case x509.ExtKeyUsageMicrosoftCommercialCodeSigning:
		return "Microsoft commercial code signing"
	case x509.ExtKeyUsageMicrosoftKernelCodeSigning:
		return "Microsoft kernel code signing"
	default:
		return ""
	}
}

func keyUsageToString(usage x509.KeyUsage) string {
	switch usage {
	case x509.KeyUsageDigitalSignature:
		return "digital signature"
	case x509.KeyUsageContentCommitment:
		return "content commitment"
	case x509.KeyUsageKeyEncipherment:
		return "key encipherment"
	case x509.KeyUsageDataEncipherment:
		return "data encipherment"
	case x509.KeyUsageKeyAgreement:
		return "key agreement"
	case x509.KeyUsageCertSign:
		return "certificate sign"
	case x509.KeyUsageCRLSign:
		return "CRL sign"
	case x509.KeyUsageEncipherOnly:
		return "encipher only"
	case x509.KeyUsageDecipherOnly:
		return "decipher only"
	default:
		return "digital signature"
	}
}

func encodedPEMFromCert(cert *x509.Certificate) string {
	pemCert := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})
	return string(pemCert)
}

func publicKeyFromCert(cert *x509.Certificate) string {
	pubKeyBytes, _ := x509.MarshalPKIXPublicKey(cert.PublicKey)
	pubKeyBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	}
	return sha1Hex(pubKeyBlock.Bytes)
}

func IsVerificationError(err error) bool {
	return strings.Contains(err.Error(), "tls: failed to verify")
}

func IsConnectionRefused(err error) bool {
	return strings.Contains(err.Error(), "connect: connection refused")
}

var loomingTreshold = time.Hour * 24 * 7 * 2 // 2 weeks

func getStatus(expires time.Time) string {
	if expires.Before(time.Now()) {
		return "expired"
	}
	timeLeft := time.Until(expires)
	if timeLeft < loomingTreshold {
		return "expires"
	}
	return "healthy"
}

func sha1Hex(b []byte) string {
	sha1Hash := sha1.Sum(b)
	return hex.EncodeToString(sha1Hash[:])
}
