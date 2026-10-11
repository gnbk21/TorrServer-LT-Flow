// Exercises the production native parsers, with the bridge instrumented by
// libFuzzer/ASan/UBSan. Native dependencies keep their pinned release build.
#include "../server/lt/lt_shim.h"
#include <cstddef>
#include <cstdint>
#include <cstdlib>
#include <string>

extern "C" int LLVMFuzzerTestOneInput(uint8_t const* data,std::size_t size) {
    if (size==0 || size>(1u<<20)) return 0;
    std::size_t length=0;
    auto* parsed=lt_parse_torrent_bytes_alloc(data,size,&length);
    if (parsed) lt_free(parsed);
    if (size<=4096) {
        std::string magnet(reinterpret_cast<char const*>(data),size);
        parsed=lt_parse_magnet_alloc(magnet.c_str(),&length);
        if (parsed) lt_free(parsed);
    }
    return 0;
}
