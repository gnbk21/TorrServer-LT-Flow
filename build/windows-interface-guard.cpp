// Explicit, reversible WFP policy for ONE executable. No automatic elevation,
// host DNS changes, shared-service filters, or claim of complete leak
// protection.
#define INITGUID
#include <winsock2.h>
#include <ws2tcpip.h>
#include <windows.h>
#include <fwpmu.h>
#include <iphlpapi.h>
#include <netioapi.h>
#include <objbase.h>
#include <algorithm>
#include <array>
#include <cstdint>
#include <cstring>
#include <iostream>
#include <stdexcept>
#include <string>
#include <vector>

static GUID providerKey = {0x4b6555c1,
                           0xf72b,
                           0x4824,
                           {0x91, 0xa4, 0x70, 0x35, 0x4b, 0x5c, 0x21, 0x6f}};
static GUID sublayerKey = {0x8d1e07d5,
                           0xd039,
                           0x4c62,
                           {0x86, 0xe2, 0xd4, 0xa1, 0xbe, 0x93, 0xf7, 0x1d}};
constexpr unsigned ruleCount = 30;
static const std::array<GUID, 4> layers = {
    FWPM_LAYER_ALE_AUTH_CONNECT_V4, FWPM_LAYER_ALE_AUTH_RECV_ACCEPT_V4,
    FWPM_LAYER_ALE_AUTH_CONNECT_V6, FWPM_LAYER_ALE_AUTH_RECV_ACCEPT_V6};
static const std::array<FWP_V4_ADDR_AND_MASK, 7> lan4 = {
    {{0x0a000000, 0xff000000},
     {0xac100000, 0xfff00000},
     {0xc0a80000, 0xffff0000},
     {0x7f000000, 0xff000000},
     {0xa9fe0000, 0xffff0000},
     {0xe0000000, 0xffffff00},
     {0xeffffffa, 0xffffffff}}};
static std::array<FWP_V6_ADDR_AND_MASK, 4> localIPv6() {
  std::array<FWP_V6_ADDR_AND_MASK, 4> ranges{};
  ranges[0].addr[15] = 1;
  ranges[0].prefixLength = 128;
  ranges[1].addr[0] = 0xfc;
  ranges[1].prefixLength = 7;
  ranges[2].addr[0] = 0xfe;
  ranges[2].addr[1] = 0x80;
  ranges[2].prefixLength = 10;
  ranges[3].addr[0] = 0xff;
  ranges[3].addr[1] = 2;
  ranges[3].prefixLength = 16;
  return ranges;
}
struct Tag {
  char magic[8];
  UINT32 rule;
  UINT32 reserved;
  UINT64 luid;
};
static_assert(sizeof(Tag) == 24, "persistent tag layout changed");
static Tag makeTag(unsigned rule, UINT64 luid) {
  Tag tag{};
  std::memcpy(tag.magic, "FLWGRD1", 8);
  tag.rule = rule;
  tag.luid = luid;
  return tag;
}
static bool tagged(const FWPM_FILTER0 &filter, Tag &tag) {
  if (!filter.providerKey || *filter.providerKey != providerKey ||
      filter.subLayerKey != sublayerKey ||
      filter.providerData.size != sizeof(tag) || !filter.providerData.data)
    return false;
  std::memcpy(&tag, filter.providerData.data, sizeof(tag));
  return std::memcmp(tag.magic, "FLWGRD1", 8) == 0 && tag.rule < ruleCount &&
         tag.reserved == 0;
}
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
  FWP_BYTE_BLOB *blob = nullptr;
  explicit App(const std::wstring &name) {
    check(FwpmGetAppIdFromFileName0(name.c_str(), &blob));
  }
  ~App() {
    if (blob)
      FwpmFreeMemory0(reinterpret_cast<void **>(&blob));
  }
};
struct Transaction {
  Engine &engine;
  bool complete = false;
  explicit Transaction(Engine &value) : engine(value) {
    check(FwpmTransactionBegin0(engine.handle, 0));
  }
  ~Transaction() {
    if (!complete)
      FwpmTransactionAbort0(engine.handle);
  }
  void commit() {
    check(FwpmTransactionCommit0(engine.handle));
    complete = true;
  }
};
static bool sameApp(const FWPM_FILTER0 &filter, const FWP_BYTE_BLOB *app) {
  if (!app)
    return true; // Only the explicit remove-all-owned command uses this.
  for (UINT32 i = 0; i < filter.numFilterConditions; i++) {
    const auto &condition = filter.filterCondition[i];
    if (condition.fieldKey == FWPM_CONDITION_ALE_APP_ID &&
        condition.matchType == FWP_MATCH_EQUAL &&
        condition.conditionValue.type == FWP_BYTE_BLOB_TYPE) {
      const auto *blob = condition.conditionValue.byteBlob;
      return blob && blob->size == app->size &&
             std::memcmp(blob->data, app->data, app->size) == 0;
    }
  }
  return false;
}
static bool intact(const FWPM_FILTER0 &filter, const Tag &tag) {
  const unsigned layer = tag.rule < 9    ? 0
                         : tag.rule < 18 ? 1
                         : tag.rule < 24 ? 2
                                         : 3;
  const std::array<unsigned, 4> bases = {0, 9, 18, 24};
  const unsigned offset = tag.rule - bases[layer];
  const unsigned last = layer < 2 ? 8 : 5;
  const bool permit = offset != last;
  if (filter.layerKey != layers[layer] ||
      !(filter.flags & FWPM_FILTER_FLAG_PERSISTENT) ||
      (filter.flags & FWPM_FILTER_FLAG_DISABLED) ||
      filter.action.type !=
          static_cast<FWP_ACTION_TYPE>(permit ? FWP_ACTION_PERMIT
                                              : FWP_ACTION_BLOCK) ||
      filter.weight.type != FWP_UINT8 ||
      filter.weight.uint8 != (permit ? 15 : 1) ||
      filter.numFilterConditions != (permit ? 2u : 1u))
    return false;
  if (!permit)
    return true;
  const FWPM_FILTER_CONDITION0 *extra = nullptr;
  for (UINT32 i = 0; i < filter.numFilterConditions; i++)
    if (filter.filterCondition[i].fieldKey != FWPM_CONDITION_ALE_APP_ID)
      extra = &filter.filterCondition[i];
  if (!extra || extra->matchType != FWP_MATCH_EQUAL)
    return false;
  const auto &value = extra->conditionValue;
  if (offset == 0)
    return extra->fieldKey == FWPM_CONDITION_IP_LOCAL_INTERFACE &&
           value.type == FWP_UINT64 && value.uint64 &&
           *value.uint64 == tag.luid;
  if (extra->fieldKey != FWPM_CONDITION_IP_REMOTE_ADDRESS)
    return false;
  if (layer < 2)
    return value.type == FWP_V4_ADDR_MASK && value.v4AddrMask &&
           value.v4AddrMask->addr == lan4[offset - 1].addr &&
           value.v4AddrMask->mask == lan4[offset - 1].mask;
  const auto ranges = localIPv6();
  return value.type == FWP_V6_ADDR_MASK && value.v6AddrMask &&
         value.v6AddrMask->prefixLength == ranges[offset - 1].prefixLength &&
         std::memcmp(value.v6AddrMask->addr, ranges[offset - 1].addr, 16) == 0;
}
struct Found {
  GUID key;
  Tag tag;
  bool intact;
};
static std::vector<Found> enumerate(Engine &engine, const FWP_BYTE_BLOB *app) {
  FWPM_FILTER_ENUM_TEMPLATE0 query{};
  query.providerKey = &providerKey;
  query.enumType = FWP_FILTER_ENUM_FULLY_CONTAINED;
  query.flags = FWP_FILTER_ENUM_FLAG_INCLUDE_DISABLED;
  query.actionMask = UINT32_MAX;
  std::vector<Found> found;
  unsigned scanned = 0;
  for (const auto &layer : layers) {
    query.layerKey = layer;
    HANDLE enumeration = nullptr;
    auto error =
        FwpmFilterCreateEnumHandle0(engine.handle, &query, &enumeration);
    if (error == FWP_E_NEVER_MATCH)
      continue;
    check(error);
    try {
      for (;;) {
        FWPM_FILTER0 **filters = nullptr;
        UINT32 count = 0;
        check(
            FwpmFilterEnum0(engine.handle, enumeration, 64, &filters, &count));
        scanned += count;
        for (UINT32 i = 0; i < count; i++) {
          Tag tag{};
          if (tagged(*filters[i], tag) && sameApp(*filters[i], app))
            found.push_back(
                {filters[i]->filterKey, tag, intact(*filters[i], tag)});
        }
        if (filters)
          FwpmFreeMemory0(reinterpret_cast<void **>(&filters));
        if (count == 0)
          break;
        if (scanned > 8192)
          check(ERROR_MORE_DATA);
      }
    } catch (...) {
      FwpmFilterDestroyEnumHandle0(engine.handle, enumeration);
      throw;
    }
    check(FwpmFilterDestroyEnumHandle0(engine.handle, enumeration));
  }
  return found;
}
static void ensureProvider(Engine &engine) {
  FWPM_PROVIDER0 provider{};
  provider.providerKey = providerKey;
  provider.displayData.name =
      const_cast<wchar_t *>(L"TorrServer-Flow interface guard");
  provider.flags = FWPM_PROVIDER_FLAG_PERSISTENT;
  auto error = FwpmProviderAdd0(engine.handle, &provider, nullptr);
  if (error != FWP_E_ALREADY_EXISTS)
    check(error);
  FWPM_SUBLAYER0 sublayer{};
  sublayer.subLayerKey = sublayerKey;
  sublayer.providerKey = &providerKey;
  sublayer.displayData.name = provider.displayData.name;
  sublayer.flags = FWPM_SUBLAYER_FLAG_PERSISTENT;
  sublayer.weight = 0x5000;
  error = FwpmSubLayerAdd0(engine.handle, &sublayer, nullptr);
  if (error != FWP_E_ALREADY_EXISTS)
    check(error);
}
static void add(Engine &engine, const App &app, const GUID &layer, UINT64 luid,
                unsigned rule, const FWP_CONDITION_VALUE0 *extra,
                const GUID *field) {
  Tag tag = makeTag(rule, luid);
  std::array<FWPM_FILTER_CONDITION0, 2> conditions{};
  conditions[0].fieldKey = FWPM_CONDITION_ALE_APP_ID;
  conditions[0].matchType = FWP_MATCH_EQUAL;
  conditions[0].conditionValue.type = FWP_BYTE_BLOB_TYPE;
  conditions[0].conditionValue.byteBlob = app.blob;
  if (extra) {
    conditions[1].fieldKey = *field;
    conditions[1].matchType = FWP_MATCH_EQUAL;
    conditions[1].conditionValue = *extra;
  }
  FWPM_FILTER0 filter{};
  check(static_cast<DWORD>(CoCreateGuid(&filter.filterKey)));
  filter.displayData.name =
      const_cast<wchar_t *>(extra ? L"Flow selected-interface or LAN permit"
                                  : L"Flow public fallback block");
  filter.providerKey = &providerKey;
  filter.subLayerKey = sublayerKey;
  filter.layerKey = layer;
  filter.flags = FWPM_FILTER_FLAG_PERSISTENT;
  filter.providerData = {sizeof(tag), reinterpret_cast<UINT8 *>(&tag)};
  filter.weight.type = FWP_UINT8;
  filter.weight.uint8 = extra ? 15 : 1;
  filter.action.type = extra ? FWP_ACTION_PERMIT : FWP_ACTION_BLOCK;
  filter.numFilterConditions = extra ? 2 : 1;
  filter.filterCondition = conditions.data();
  check(FwpmFilterAdd0(engine.handle, &filter, nullptr, nullptr));
}
static void install(Engine &engine, const App &app, UINT64 luid) {
  Transaction transaction(engine);
  ensureProvider(engine);
  for (const auto &old : enumerate(engine, app.blob))
    check(FwpmFilterDeleteByKey0(engine.handle, &old.key));
  // IP values/masks are host order, as required by FWP_V4_ADDR_AND_MASK.
  const auto lan6 = localIPv6();
  unsigned rule = 0;
  for (unsigned i = 0; i < layers.size(); i++) {
    FWP_CONDITION_VALUE0 selected{};
    selected.type = FWP_UINT64;
    selected.uint64 = &luid;
    add(engine, app, layers[i], luid, rule++, &selected,
        &FWPM_CONDITION_IP_LOCAL_INTERFACE);
    if (i < 2) {
      for (auto range : lan4) {
        FWP_CONDITION_VALUE0 value{};
        value.type = FWP_V4_ADDR_MASK;
        value.v4AddrMask = &range;
        add(engine, app, layers[i], luid, rule++, &value,
            &FWPM_CONDITION_IP_REMOTE_ADDRESS);
      }
    } else {
      for (auto range : lan6) {
        FWP_CONDITION_VALUE0 value{};
        value.type = FWP_V6_ADDR_MASK;
        value.v6AddrMask = &range;
        add(engine, app, layers[i], luid, rule++, &value,
            &FWPM_CONDITION_IP_REMOTE_ADDRESS);
      }
    }
    add(engine, app, layers[i], luid, rule++, nullptr, nullptr);
  }
  if (rule != ruleCount)
    check(ERROR_INVALID_DATA);
  transaction.commit();
}
static void status(Engine &engine, const App &app) {
  auto filters = enumerate(engine, app.blob);
  UINT64 luid = filters.empty() ? 0 : filters[0].tag.luid;
  std::array<bool, ruleCount> rules{};
  bool complete = filters.size() == ruleCount;
  for (const auto &filter : filters) {
    if (!filter.intact || filter.tag.luid != luid || rules[filter.tag.rule])
      complete = false;
    rules[filter.tag.rule] = true;
  }
  complete = complete && std::all_of(rules.begin(), rules.end(),
                                     [](bool value) { return value; });
  std::cout << "{\"installed\":" << (filters.empty() ? "false" : "true")
            << ",\"configuration_complete\":" << (complete ? "true" : "false")
            << ",\"filter_count\":" << filters.size()
            << ",\"interface_luid\":" << luid
            << ",\"dns_verified\":false,\"leak_protection_verified\":false,"
               "\"scope\":\"Selected executable ALE IPv4/IPv6 connections; LAN "
               "exceptions; shared DNS is outside this policy\"}\n";
}
static std::wstring absolutePath(const std::wstring &path) {
  std::vector<wchar_t> value(32768);
  DWORD length = GetFullPathNameW(
      path.c_str(), static_cast<DWORD>(value.size()), value.data(), nullptr);
  if (!length || length >= value.size())
    check(ERROR_INVALID_NAME);
  return std::wstring(value.data(), length);
}
int wmain(int argc, wchar_t **argv) {
  try {
    if (argc < 2)
      check(ERROR_BAD_ARGUMENTS);
    std::wstring mode = argv[1], name;
    ULONG index = 0;
    for (int i = 2; i < argc; i++) {
      const std::wstring key = argv[i];
      if (key == L"--exe" && i + 1 < argc)
        name = absolutePath(argv[++i]);
      else if (key == L"--interface-index" && i + 1 < argc) {
        const std::wstring text = argv[++i];
        size_t consumed = 0;
        const auto number = std::stoull(text, &consumed);
        if (consumed != text.size() || number == 0 || number > UINT32_MAX ||
            text[0] == L'-')
          check(ERROR_BAD_ARGUMENTS);
        index = static_cast<ULONG>(number);
      } else
        check(ERROR_BAD_ARGUMENTS);
    }
    if (mode == L"remove-all-owned") {
      if (!name.empty() || index)
        check(ERROR_BAD_ARGUMENTS);
      Engine engine;
      Transaction transaction(engine);
      const auto filters = enumerate(engine, nullptr);
      for (const auto &filter : filters)
        check(FwpmFilterDeleteByKey0(engine.handle, &filter.key));
      transaction.commit();
      std::cout << "{\"removed_filters\":" << filters.size() << "}\n";
      return 0;
    }
    if (name.empty() || (mode != L"preview" && mode != L"install" &&
                         mode != L"status" && mode != L"remove"))
      check(ERROR_BAD_ARGUMENTS);
    App app(name);
    if (mode == L"preview" || mode == L"install") {
      if (!index)
        check(ERROR_BAD_ARGUMENTS);
      const auto attributes = GetFileAttributesW(name.c_str());
      if (attributes == INVALID_FILE_ATTRIBUTES ||
          (attributes &
           (FILE_ATTRIBUTE_DIRECTORY | FILE_ATTRIBUTE_REPARSE_POINT)))
        check(ERROR_FILE_NOT_FOUND);
      NET_LUID luid{};
      check(ConvertInterfaceIndexToLuid(index, &luid));
      MIB_IF_ROW2 row{};
      row.InterfaceLuid = luid;
      check(GetIfEntry2(&row));
      if (row.OperStatus != IfOperStatusUp)
        check(ERROR_NETWORK_UNREACHABLE);
      if (mode == L"preview") {
        std::cout << "{\"changes_applied\":false,\"planned_filters\":"
                  << ruleCount << ",\"interface_luid\":" << luid.Value
                  << ",\"persistent\":true,\"dns_verified\":false,\"leak_"
                     "protection_verified\":false}\n";
        return 0;
      }
      Engine engine;
      install(engine, app, luid.Value);
      status(engine, app);
      return 0;
    }
    if (index)
      check(ERROR_BAD_ARGUMENTS);
    Engine engine;
    if (mode == L"status") {
      status(engine, app);
      return 0;
    }
    Transaction transaction(engine);
    const auto filters = enumerate(engine, app.blob);
    for (const auto &filter : filters)
      check(FwpmFilterDeleteByKey0(engine.handle, &filter.key));
    transaction.commit();
    std::cout << "{\"removed_filters\":" << filters.size() << "}\n";
    return 0;
  } catch (DWORD code) {
    // Error numbers are useful, bounded and cannot disclose executable paths.
    std::cerr
        << "{\"error\":\"Windows interface guard operation failed\",\"code\":"
        << code << "}\n";
    return 1;
  } catch (...) {
    std::cerr
        << "{\"error\":\"Use preview/install --exe PATH --interface-index N; "
           "status/remove --exe PATH; or remove-all-owned\"}\n";
    return 1;
  }
}
