package shims

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"hash"
)

func digestHash(algorithm string) hash.Hash {
	switch algorithm {
	case "MD5":
		return md5.New()
	case "SHA1":
		return sha1.New()
	case "SHA256":
		return sha256.New()
	case "SHA384":
		return sha512.New384()
	case "SHA512":
		return sha512.New()
	default:
		return sha256.New()
	}
}

func digestCompute(algorithm, data string) []byte {
	h := digestHash(algorithm)
	h.Write([]byte(data))
	return h.Sum(nil)
}

func DigestMD5Hexdigest(data string) string {
	return hex.EncodeToString(digestCompute("MD5", data))
}

func DigestMD5Digest(data string) string {
	return string(digestCompute("MD5", data))
}

func DigestMD5Base64digest(data string) string {
	return base64.StdEncoding.EncodeToString(digestCompute("MD5", data))
}

func DigestSHA1Hexdigest(data string) string {
	return hex.EncodeToString(digestCompute("SHA1", data))
}

func DigestSHA1Digest(data string) string {
	return string(digestCompute("SHA1", data))
}

func DigestSHA1Base64digest(data string) string {
	return base64.StdEncoding.EncodeToString(digestCompute("SHA1", data))
}

func DigestSHA256Hexdigest(data string) string {
	return hex.EncodeToString(digestCompute("SHA256", data))
}

func DigestSHA256Digest(data string) string {
	return string(digestCompute("SHA256", data))
}

func DigestSHA256Base64digest(data string) string {
	return base64.StdEncoding.EncodeToString(digestCompute("SHA256", data))
}

func DigestSHA384Hexdigest(data string) string {
	return hex.EncodeToString(digestCompute("SHA384", data))
}

func DigestSHA384Digest(data string) string {
	return string(digestCompute("SHA384", data))
}

func DigestSHA384Base64digest(data string) string {
	return base64.StdEncoding.EncodeToString(digestCompute("SHA384", data))
}

func DigestSHA512Hexdigest(data string) string {
	return hex.EncodeToString(digestCompute("SHA512", data))
}

func DigestSHA512Digest(data string) string {
	return string(digestCompute("SHA512", data))
}

func DigestSHA512Base64digest(data string) string {
	return base64.StdEncoding.EncodeToString(digestCompute("SHA512", data))
}
