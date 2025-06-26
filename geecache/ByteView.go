package geecache

// ByteView 是一个只读的字节视图，用于保证缓存值的不可变性。
// 它可以持有任意类型的数据（例如字符串或图片），但其内容一旦创建便不能被修改。
type ByteView struct {
	b []byte // b 是一个字节切片，用于存储实际数据。它被视为只读。
}

// Len 实现了 lru.Value 接口，返回 ByteView 所持有的数据的字节长度。
//
// 返回值:
//
//	int: 数据的字节长度。
func (v ByteView) Len() int {
	return len(v.b)
}

// ByteSlice 返回一个数据的拷贝。
//
// 为了保证 ByteView 的不可变性，此方法返回一个底层字节数组的克隆，
// 防止外部代码通过切片修改原始数据。
//
// 返回值:
//
//	[]byte: 数据的安全拷贝。
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

// String 将数据作为字符串返回，并实现了 fmt.Stringer 接口。
//
// 返回值:
//
//	string: 数据的字符串表示。
func (v ByteView) String() string {
	return string(v.b)
}

// cloneBytes 创建并返回一个字节切片的拷贝。
//
// 这是一个内部辅助函数，用于在创建 ByteView 或返回其内容时
// 复制源数据，以确保 ByteView 的不可变性。
//
// 参数:
//
//	bytes: 源字节切片。
//
// 返回值:
//
//	[]byte: 源字节切片的精确拷贝。
func cloneBytes(bytes []byte) []byte {
	newBytes := make([]byte, len(bytes))
	copy(newBytes, bytes)
	return newBytes
}
