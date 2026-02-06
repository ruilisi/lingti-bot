package wecom

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"math/rand"
	"sort"
	"strings"
)

// MsgCrypt handles WeChat Work message encryption/decryption
type MsgCrypt struct {
	token          string
	encodingAESKey string
	corpID         string
	aesKey         []byte
}

// NewMsgCrypt creates a new message cryptographer
func NewMsgCrypt(token, encodingAESKey, corpID string) (*MsgCrypt, error) {
	if len(encodingAESKey) != 43 {
		return nil, fmt.Errorf("encodingAESKey must be 43 characters")
	}

	aesKey, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return nil, fmt.Errorf("invalid encodingAESKey: %w", err)
	}

	return &MsgCrypt{
		token:          token,
		encodingAESKey: encodingAESKey,
		corpID:         corpID,
		aesKey:         aesKey,
	}, nil
}

// EncryptedMsg represents the encrypted message XML structure
type EncryptedMsg struct {
	XMLName    xml.Name `xml:"xml"`
	ToUserName string   `xml:"ToUserName"`
	Encrypt    string   `xml:"Encrypt"`
	AgentID    string   `xml:"AgentID,omitempty"`
}

// VerifyURL verifies the callback URL during configuration
// Returns the decrypted echostr if verification succeeds
func (mc *MsgCrypt) VerifyURL(msgSignature, timestamp, nonce, echostr string) (string, error) {
	// Calculate signature
	signature := mc.calcSignature(timestamp, nonce, echostr)
	if signature != msgSignature {
		return "", fmt.Errorf("signature verification failed")
	}

	// Decrypt echostr
	plaintext, err := mc.decrypt(echostr)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt echostr: %w", err)
	}

	return plaintext, nil
}

// DecryptMsg decrypts an incoming message
func (mc *MsgCrypt) DecryptMsg(msgSignature, timestamp, nonce string, encryptedMsg *EncryptedMsg) ([]byte, error) {
	// Verify signature
	signature := mc.calcSignature(timestamp, nonce, encryptedMsg.Encrypt)
	if signature != msgSignature {
		return nil, fmt.Errorf("signature verification failed")
	}

	// Decrypt message
	plaintext, err := mc.decrypt(encryptedMsg.Encrypt)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt message: %w", err)
	}

	return []byte(plaintext), nil
}

// EncryptMsg encrypts an outgoing message
func (mc *MsgCrypt) EncryptMsg(replyMsg, timestamp, nonce string) (string, string, error) {
	// Encrypt the message
	encrypted, err := mc.encrypt(replyMsg)
	if err != nil {
		return "", "", fmt.Errorf("failed to encrypt message: %w", err)
	}

	// Calculate signature
	signature := mc.calcSignature(timestamp, nonce, encrypted)

	return encrypted, signature, nil
}

// calcSignature calculates the message signature
func (mc *MsgCrypt) calcSignature(timestamp, nonce string, encrypted string) string {
	strs := []string{mc.token, timestamp, nonce, encrypted}
	sort.Strings(strs)
	joined := strings.Join(strs, "")

	hash := sha1.New()
	hash.Write([]byte(joined))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

// decrypt decrypts the encrypted message using AES-256-CBC
func (mc *MsgCrypt) decrypt(encrypted string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("base64 decode failed: %w", err)
	}

	block, err := aes.NewCipher(mc.aesKey)
	if err != nil {
		return "", fmt.Errorf("create cipher failed: %w", err)
	}

	if len(ciphertext) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	iv := mc.aesKey[:aes.BlockSize]
	mode := cipher.NewCBCDecrypter(block, iv)

	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	// Remove PKCS7 padding
	plaintext, err = pkcs7Unpad(plaintext)
	if err != nil {
		return "", fmt.Errorf("unpad failed: %w", err)
	}

	// Parse the plaintext: random(16) + msgLen(4) + msg + corpID
	if len(plaintext) < 20 {
		return "", fmt.Errorf("plaintext too short")
	}

	msgLen := binary.BigEndian.Uint32(plaintext[16:20])
	if uint32(len(plaintext)) < 20+msgLen {
		return "", fmt.Errorf("invalid message length")
	}

	msg := string(plaintext[20 : 20+msgLen])
	corpID := string(plaintext[20+msgLen:])

	if corpID != mc.corpID {
		return "", fmt.Errorf("corpID mismatch: expected %s, got %s", mc.corpID, corpID)
	}

	return msg, nil
}

// encrypt encrypts the message using AES-256-CBC
func (mc *MsgCrypt) encrypt(plaintext string) (string, error) {
	// Build the full message: random(16) + msgLen(4) + msg + corpID
	randomBytes := make([]byte, 16)
	for i := range randomBytes {
		randomBytes[i] = byte(rand.Intn(256))
	}

	msgBytes := []byte(plaintext)
	msgLen := make([]byte, 4)
	binary.BigEndian.PutUint32(msgLen, uint32(len(msgBytes)))

	fullMsg := bytes.Join([][]byte{
		randomBytes,
		msgLen,
		msgBytes,
		[]byte(mc.corpID),
	}, nil)

	// PKCS7 padding (WeCom uses 32-byte block size)
	fullMsg = pkcs7Pad(fullMsg, wecomBlockSize)

	block, err := aes.NewCipher(mc.aesKey)
	if err != nil {
		return "", fmt.Errorf("create cipher failed: %w", err)
	}

	iv := mc.aesKey[:aes.BlockSize]
	mode := cipher.NewCBCEncrypter(block, iv)

	ciphertext := make([]byte, len(fullMsg))
	mode.CryptBlocks(ciphertext, fullMsg)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// wecomBlockSize is the block size used by WeCom for PKCS7 padding
// Note: WeCom uses 32 bytes, not the standard AES block size of 16
const wecomBlockSize = 32

// pkcs7Pad adds PKCS7 padding
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

// pkcs7Unpad removes PKCS7 padding (using WeCom's 32-byte block size)
func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	if len(data)%wecomBlockSize != 0 {
		return nil, fmt.Errorf("data length not multiple of block size")
	}
	padding := int(data[len(data)-1])
	if padding > len(data) || padding > wecomBlockSize {
		return nil, fmt.Errorf("invalid padding")
	}
	return data[:len(data)-padding], nil
}
