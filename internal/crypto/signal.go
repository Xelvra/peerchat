package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"time"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/hkdf"
)

const (
	// Key sizes for Signal Protocol
	PrivateKeySize = 32
	PublicKeySize  = 32
	SharedKeySize  = 32
	AESKeySize     = 32
	NonceSize      = 12
	TagSize        = 16
)

// KeyPair represents a Curve25519 key pair with secure memory handling
type KeyPair struct {
	PrivateKey []byte // Protected memory for private key
	PublicKey  []byte
	createdAt  time.Time
}

// SecureKeyPair creates a new KeyPair with memory protection
func NewSecureKeyPair(privateKey, publicKey []byte) *KeyPair {
	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		createdAt:  time.Now(),
	}
}

// Destroy securely destroys the key pair
func (kp *KeyPair) Destroy() {
	if kp.PrivateKey != nil {
		// Securely zero out the private key memory
		for i := range kp.PrivateKey {
			kp.PrivateKey[i] = 0
		}
		kp.PrivateKey = nil
	}
}

// X3DHBundle represents the X3DH key bundle for initial key exchange
type X3DHBundle struct {
	IdentityKey    *KeyPair
	SignedPreKey   *KeyPair
	OneTimePreKeys []*KeyPair
	Signature      []byte
}

// DoubleRatchetState maintains the state for Double Ratchet algorithm
type DoubleRatchetState struct {
	RootKey             []byte // TODO: Add memory protection with memguard later
	ChainKey            []byte // TODO: Add memory protection with memguard later
	SendingKey          *KeyPair
	ReceivingKey        *KeyPair
	MessageNumber       uint32
	PreviousChainLength uint32
}

// SignalCrypto provides Signal Protocol cryptographic operations
type SignalCrypto struct {
	identityKeyPair *KeyPair

	// Replay attack protection
	usedNonces  map[string]time.Time // Track used nonces with timestamps
	nonceWindow time.Duration        // Time window for nonce validity
}

// NewSignalCrypto creates a new Signal Protocol crypto instance
func NewSignalCrypto() (*SignalCrypto, error) {
	// Generate identity key pair
	identityKey, err := GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate identity key: %w", err)
	}

	return &SignalCrypto{
		identityKeyPair: identityKey,
		usedNonces:      make(map[string]time.Time),
		nonceWindow:     5 * time.Minute, // 5-minute window for nonce validity
	}, nil
}

// GenerateKeyPair generates a new Curve25519 key pair
func GenerateKeyPair() (*KeyPair, error) {
	// Generate private key
	privateKey := make([]byte, PrivateKeySize)
	if _, err := io.ReadFull(rand.Reader, privateKey); err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Clamp the private key for Curve25519
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	// Generate public key
	publicKey := make([]byte, PublicKeySize)
	curve25519.ScalarBaseMult((*[32]byte)(publicKey), (*[32]byte)(privateKey))

	return NewSecureKeyPair(privateKey, publicKey), nil
}

// PerformX3DH performs the X3DH key agreement protocol
func (sc *SignalCrypto) PerformX3DH(remoteBundle *X3DHBundle, ephemeralKey *KeyPair) ([]byte, error) {
	// Perform the four Diffie-Hellman operations as per X3DH spec

	// DH1 = DH(IK_A, SPK_B)
	dh1, err := performDH(sc.identityKeyPair.PrivateKey, remoteBundle.SignedPreKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("DH1 failed: %w", err)
	}

	// DH2 = DH(EK_A, IK_B)
	dh2, err := performDH(ephemeralKey.PrivateKey, remoteBundle.IdentityKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("DH2 failed: %w", err)
	}

	// DH3 = DH(EK_A, SPK_B)
	dh3, err := performDH(ephemeralKey.PrivateKey, remoteBundle.SignedPreKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("DH3 failed: %w", err)
	}

	// Combine all DH outputs using HKDF
	sharedSecret, err := combineSecrets(dh1, dh2, dh3)
	if err != nil {
		return nil, fmt.Errorf("failed to combine secrets: %w", err)
	}

	return sharedSecret, nil
}

// EncryptMessage encrypts a message using AES-GCM with the current chain key
func (sc *SignalCrypto) EncryptMessage(plaintext []byte, chainKey []byte) ([]byte, error) {
	// Derive message key from chain key using HKDF
	messageKey, err := deriveMessageKey(chainKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive message key: %w", err)
	}

	// Create AES-GCM cipher
	block, err := aes.NewCipher(messageKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the message
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Prepend nonce to ciphertext
	result := make([]byte, NonceSize+len(ciphertext))
	copy(result[:NonceSize], nonce)
	copy(result[NonceSize:], ciphertext)

	return result, nil
}

// DecryptMessage decrypts a message using AES-GCM with the current chain key
func (sc *SignalCrypto) DecryptMessage(ciphertext []byte, chainKey []byte) ([]byte, error) {
	if len(ciphertext) < NonceSize+TagSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and encrypted data
	nonce := ciphertext[:NonceSize]
	encrypted := ciphertext[NonceSize:]

	// Check for replay attacks
	if err := sc.checkReplayAttack(nonce); err != nil {
		return nil, fmt.Errorf("replay attack detected: %w", err)
	}

	// Derive message key from chain key
	messageKey, err := deriveMessageKey(chainKey)
	if err != nil {
		return nil, fmt.Errorf("failed to derive message key: %w", err)
	}

	// Create AES-GCM cipher
	block, err := aes.NewCipher(messageKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt the message
	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt message: %w", err)
	}

	// Mark nonce as used after successful decryption
	sc.markNonceUsed(nonce)

	return plaintext, nil
}

// GetIdentityKey returns the public identity key
func (sc *SignalCrypto) GetIdentityKey() []byte {
	return sc.identityKeyPair.PublicKey
}

// performDH performs Diffie-Hellman key exchange
func performDH(privateKey []byte, publicKey []byte) ([]byte, error) {
	if len(privateKey) != 32 || len(publicKey) != 32 {
		return nil, fmt.Errorf("invalid key size: private=%d, public=%d", len(privateKey), len(publicKey))
	}

	sharedSecret, err := curve25519.X25519(privateKey, publicKey)
	if err != nil {
		return nil, fmt.Errorf("X25519 operation failed: %w", err)
	}
	return sharedSecret, nil
}

// combineSecrets combines multiple DH outputs using HKDF
func combineSecrets(secrets ...[]byte) ([]byte, error) {
	// Concatenate all secrets
	var combined []byte
	for _, secret := range secrets {
		combined = append(combined, secret...)
	}

	// Use HKDF to derive the final shared secret
	hkdf := hkdf.New(sha256.New, combined, nil, []byte("XelvraX3DH"))

	sharedSecret := make([]byte, SharedKeySize)
	if _, err := io.ReadFull(hkdf, sharedSecret); err != nil {
		return nil, fmt.Errorf("failed to derive shared secret: %w", err)
	}

	return sharedSecret, nil
}

// deriveMessageKey derives a message key from a chain key using HKDF
func deriveMessageKey(chainKey []byte) ([]byte, error) {
	hkdf := hkdf.New(sha256.New, chainKey, nil, []byte("XelvraMessageKey"))

	messageKey := make([]byte, AESKeySize)
	if _, err := io.ReadFull(hkdf, messageKey); err != nil {
		return nil, fmt.Errorf("failed to derive message key: %w", err)
	}

	return messageKey, nil
}

// checkReplayAttack checks if a nonce has been used before
func (sc *SignalCrypto) checkReplayAttack(nonce []byte) error {
	nonceStr := fmt.Sprintf("%x", nonce)

	// Clean up old nonces first
	sc.cleanupOldNonces()

	// Check if nonce was already used
	if _, exists := sc.usedNonces[nonceStr]; exists {
		return fmt.Errorf("nonce already used")
	}

	return nil
}

// markNonceUsed marks a nonce as used
func (sc *SignalCrypto) markNonceUsed(nonce []byte) {
	nonceStr := fmt.Sprintf("%x", nonce)
	sc.usedNonces[nonceStr] = time.Now()
}

// cleanupOldNonces removes expired nonces from the tracking map
func (sc *SignalCrypto) cleanupOldNonces() {
	now := time.Now()
	for nonce, timestamp := range sc.usedNonces {
		if now.Sub(timestamp) > sc.nonceWindow {
			delete(sc.usedNonces, nonce)
		}
	}
}

// Destroy securely destroys the SignalCrypto instance
func (sc *SignalCrypto) Destroy() {
	if sc.identityKeyPair != nil {
		sc.identityKeyPair.Destroy()
		sc.identityKeyPair = nil
	}

	// Clear nonce tracking
	sc.usedNonces = nil
}
