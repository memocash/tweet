package db

type Block struct {
	BlockHash string
	Block     []byte
}

func (b *Block) GetPrefix() string {
	return PrefixBlock
}

func (b *Block) GetUid() []byte {
	return []byte(b.BlockHash)
}

func (b *Block) SetUid(u []byte) {
	b.BlockHash = string(u)
}

func (b *Block) Serialize() []byte {
	return b.Block
}

func (b *Block) Deserialize(d []byte) {
	b.Block = d
}
