# TODO
- [v] implement Kirsch–Mitzenmacher optimization
- [v] implement persist and load from persist (serialize / deserialize)
- [ ] implement compression in the persistence
- [ ] implement thread safety

Thread safety isn't documented. Concurrent Add calls race on the same byte — read-modify-write on |= is not atomic, so a concurrent set can be lost, producing a false negative, which breaks the filter's one hard guarantee. Either document "not safe for concurrent use" or add an RWMutex.


https://maltsev.space/blog/008-bloom-filters-pt1

serialize with std lib binary (gob) use io.Reader/writer to abstract the particular reader and writer (network/file)
// add options for withCompression, withFile, withCustomReaderWriter
