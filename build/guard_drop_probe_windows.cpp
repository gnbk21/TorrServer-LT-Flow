// CI-only, read-only evidence for asynchronous UDP blocking. Never packaged.
// Enumerate only the generated application's events in the requested time
// range; export counts only after identifying a block installed by the Flow
// provider.
#define INITGUID
#include <winsock2.h>
#include <windows.h>
#include <fwpmu.h>
#include <array>
#include <cstring>
#include <iostream>
#include <string>

static const GUID provider = {0x4b6555c1,
                              0xf72b,
                              0x4824,
                              {0x91, 0xa4, 0x70, 0x35, 0x4b, 0x5c, 0x21, 0x6f}};
static const GUID sublayer = {0x8d1e07d5,
                              0xd039,
                              0x4c62,
                              {0x86, 0xe2, 0xd4, 0xa1, 0xbe, 0x93, 0xf7, 0x1d}};
static void check(DWORD code) {
  if (code != ERROR_SUCCESS)
    throw code;
}
struct Engine {
  HANDLE handle = nullptr;
  Engine() {
    check(
        FwpmEngineOpen0(nullptr, RPC_C_AUTHN_WINNT, nullptr, nullptr, &handle));
  }
  ~Engine() {
    if (handle)
      FwpmEngineClose0(handle);
  }
};
struct App {
  FWP_BYTE_BLOB *value = nullptr;
  explicit App(const wchar_t *path) {
    check(FwpmGetAppIdFromFileName0(path, &value));
  }
  ~App() {
    if (value)
      FwpmFreeMemory0(reinterpret_cast<void **>(&value));
  }
};
struct Enumeration {
  Engine &engine;
  HANDLE handle = nullptr;
  Enumeration(Engine &owner, FWPM_NET_EVENT_ENUM_TEMPLATE0 &query)
      : engine(owner) {
    check(FwpmNetEventCreateEnumHandle0(engine.handle, &query, &handle));
  }
  ~Enumeration() {
    if (handle)
      FwpmNetEventDestroyEnumHandle0(engine.handle, handle);
  }
};
static bool ownedBlock(Engine &engine, UINT64 id, const App &app) {
  FWPM_FILTER0 *filter = nullptr;
  const auto result = FwpmFilterGetById0(engine.handle, id, &filter);
  if (result == static_cast<DWORD>(FWP_E_FILTER_NOT_FOUND))
    return false;
  check(result);
  bool matches = filter->providerKey && *filter->providerKey == provider &&
                 filter->subLayerKey == sublayer &&
                 filter->action.type == FWP_ACTION_BLOCK &&
                 filter->providerData.size == 24 && filter->providerData.data &&
                 std::memcmp(filter->providerData.data, "FLWGRD1", 8) == 0;
  bool exactApp = false;
  for (UINT32 i = 0; i < filter->numFilterConditions; ++i) {
    const auto &c = filter->filterCondition[i];
    if (c.fieldKey != FWPM_CONDITION_ALE_APP_ID ||
        c.matchType != FWP_MATCH_EQUAL ||
        c.conditionValue.type != FWP_BYTE_BLOB_TYPE)
      continue;
    const auto *blob = c.conditionValue.byteBlob;
    exactApp = blob && blob->size == app.value->size && blob->data &&
               std::memcmp(blob->data, app.value->data, blob->size) == 0;
  }
  FwpmFreeMemory0(reinterpret_cast<void **>(&filter));
  return matches && exactApp;
}
static bool environment(const wchar_t *name, const wchar_t *expected) {
  wchar_t value[64]{};
  const auto length = GetEnvironmentVariableW(name, value, 64);
  return length > 0 && length < 64 && std::wstring(value) == expected;
}
int wmain(int argc, wchar_t **argv) {
  try {
    // Reading unrelated workstation diagnostics is deliberately unavailable.
    if (argc != 3 || !environment(L"GITHUB_ACTIONS", L"true") ||
        !environment(L"FLOW_ISOLATED_WFP", L"github-hosted") ||
        !environment(L"RUNNER_ENVIRONMENT", L"github-hosted"))
      check(ERROR_ACCESS_DENIED);
    size_t consumed = 0;
    const std::wstring timestamp = argv[2];
    const UINT64 start = std::stoull(timestamp, &consumed);
    FILETIME end{};
    GetSystemTimeAsFileTime(&end);
    const UINT64 now =
        (static_cast<UINT64>(end.dwHighDateTime) << 32) | end.dwLowDateTime;
    if (consumed != timestamp.size() || timestamp.empty() ||
        timestamp[0] == L'-' || start > now ||
        now - start > 30ULL * 10000000ULL)
      check(ERROR_BAD_ARGUMENTS);
    Engine engine;
    App app(argv[1]);
    FWP_VALUE0 *option = nullptr;
    check(FwpmEngineGetOption0(engine.handle, FWPM_ENGINE_COLLECT_NET_EVENTS,
                               &option));
    const bool enabled =
        option && option->type == FWP_UINT32 && option->uint32 == 1;
    if (option)
      FwpmFreeMemory0(reinterpret_cast<void **>(&option));
    if (!enabled)
      check(static_cast<DWORD>(FWP_E_NET_EVENTS_DISABLED));
    FWPM_FILTER_CONDITION0 condition{};
    condition.fieldKey = FWPM_CONDITION_ALE_APP_ID;
    condition.matchType = FWP_MATCH_EQUAL;
    condition.conditionValue.type = FWP_BYTE_BLOB_TYPE;
    condition.conditionValue.byteBlob = app.value;
    FWPM_NET_EVENT_ENUM_TEMPLATE0 query{};
    query.startTime = {static_cast<DWORD>(start),
                       static_cast<DWORD>(start >> 32)};
    query.endTime = end;
    query.numFilterConditions = 1;
    query.filterCondition = &condition;
    Enumeration enumeration(engine, query);
    std::array<unsigned, 4> counts{};
    unsigned scanned = 0;
    for (;;) {
      FWPM_NET_EVENT0 **events = nullptr;
      UINT32 count = 0;
      check(FwpmNetEventEnum0(engine.handle, enumeration.handle, 32, &events,
                              &count));
      try {
        for (UINT32 i = 0; i < count; ++i) {
          if (++scanned > 256)
            check(ERROR_MORE_DATA);
          const auto &e = *events[i];
          const auto &h = e.header;
          constexpr UINT32 required = FWPM_NET_EVENT_FLAG_IP_PROTOCOL_SET |
                                      FWPM_NET_EVENT_FLAG_IP_VERSION_SET |
                                      FWPM_NET_EVENT_FLAG_REMOTE_ADDR_SET |
                                      FWPM_NET_EVENT_FLAG_REMOTE_PORT_SET |
                                      FWPM_NET_EVENT_FLAG_APP_ID_SET;
          if (e.type != FWPM_NET_EVENT_TYPE_CLASSIFY_DROP || !e.classifyDrop ||
              (h.flags & required) != required || h.remotePort != 49191 ||
              h.appId.size != app.value->size || !h.appId.data ||
              std::memcmp(h.appId.data, app.value->data, h.appId.size) != 0)
            continue;
          const std::array<UINT8, 16> remote6 = {
              0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1};
          const bool ipv4 =
              h.ipVersion == FWP_IP_VERSION_V4 && h.remoteAddrV4 == 0xc0000201;
          const bool ipv6 = h.ipVersion == FWP_IP_VERSION_V6 &&
                            std::memcmp(h.remoteAddrV6.byteArray16,
                                        remote6.data(), remote6.size()) == 0;
          if ((!ipv4 && !ipv6) ||
              (h.ipProtocol != IPPROTO_TCP && h.ipProtocol != IPPROTO_UDP) ||
              !ownedBlock(engine, e.classifyDrop->filterId, app))
            continue;
          ++counts[(ipv6 ? 2 : 0) + (h.ipProtocol == IPPROTO_UDP ? 1 : 0)];
        }
      } catch (...) {
        if (events)
          FwpmFreeMemory0(reinterpret_cast<void **>(&events));
        throw;
      }
      if (events)
        FwpmFreeMemory0(reinterpret_cast<void **>(&events));
      if (count < 32)
        break;
    }
    std::cout << "{\"tcp4\":" << counts[0] << ",\"udp4\":" << counts[1]
              << ",\"tcp6\":" << counts[2] << ",\"udp6\":" << counts[3]
              << "}\n";
    return 0;
  } catch (DWORD code) {
    std::cerr << "{\"error\":\"Scoped CI drop evidence unavailable\",\"code\":"
              << code << "}\n";
  } catch (...) {
    std::cerr << "{\"error\":\"Invalid scoped CI evidence arguments\"}\n";
  }
  return 1;
}
