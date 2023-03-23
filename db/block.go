package db

type Block struct {
	BlockHash [32]byte
	Block     []byte
}

func (b *Block) GetPrefix() string {
	return PrefixBlock
}

func (b *Block) GetUid() []byte {
	return b.BlockHash[:]
}

func (b *Block) SetUid(u []byte) {
	copy(b.BlockHash[:], u[:32])
}

func (b *Block) Serialize() []byte {
	return b.Block
}

func (b *Block) Deserialize(d []byte) {
	b.Block = d
}
