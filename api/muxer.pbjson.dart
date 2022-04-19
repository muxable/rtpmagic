///
//  Generated code. Do not modify.
//  source: muxer.proto
//
// @dart = 2.12
// ignore_for_file: annotate_overrides,camel_case_types,unnecessary_const,non_constant_identifier_names,library_prefixes,unused_import,unused_shown_name,return_of_invalid_type,unnecessary_this,prefer_final_fields,deprecated_member_use_from_same_package

import 'dart:core' as $core;
import 'dart:convert' as $convert;
import 'dart:typed_data' as $typed_data;
@$core.Deprecated('Use wifiConnectRequestDescriptor instead')
const WifiConnectRequest$json = const {
  '1': 'WifiConnectRequest',
  '2': const [
    const {'1': 'interface_id', '3': 1, '4': 1, '5': 9, '10': 'interfaceId'},
    const {'1': 'bssid', '3': 2, '4': 1, '5': 9, '10': 'bssid'},
    const {'1': 'password', '3': 3, '4': 1, '5': 9, '10': 'password'},
    const {'1': 'ap_mode_ssid', '3': 4, '4': 1, '5': 9, '10': 'apModeSsid'},
    const {'1': 'autoconnect', '3': 5, '4': 1, '5': 8, '10': 'autoconnect'},
  ],
};

/// Descriptor for `WifiConnectRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List wifiConnectRequestDescriptor = $convert.base64Decode('ChJXaWZpQ29ubmVjdFJlcXVlc3QSIQoMaW50ZXJmYWNlX2lkGAEgASgJUgtpbnRlcmZhY2VJZBIUCgVic3NpZBgCIAEoCVIFYnNzaWQSGgoIcGFzc3dvcmQYAyABKAlSCHBhc3N3b3JkEiAKDGFwX21vZGVfc3NpZBgEIAEoCVIKYXBNb2RlU3NpZBIgCgthdXRvY29ubmVjdBgFIAEoCFILYXV0b2Nvbm5lY3Q=');
@$core.Deprecated('Use wifiConnectResponseDescriptor instead')
const WifiConnectResponse$json = const {
  '1': 'WifiConnectResponse',
  '2': const [
    const {'1': 'error', '3': 1, '4': 1, '5': 9, '10': 'error'},
    const {'1': 'wifi_state', '3': 2, '4': 1, '5': 11, '6': '.api.WifiState', '10': 'wifiState'},
  ],
};

/// Descriptor for `WifiConnectResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List wifiConnectResponseDescriptor = $convert.base64Decode('ChNXaWZpQ29ubmVjdFJlc3BvbnNlEhQKBWVycm9yGAEgASgJUgVlcnJvchItCgp3aWZpX3N0YXRlGAIgASgLMg4uYXBpLldpZmlTdGF0ZVIJd2lmaVN0YXRl');
@$core.Deprecated('Use wifiStateDescriptor instead')
const WifiState$json = const {
  '1': 'WifiState',
  '2': const [
    const {'1': 'interfaces', '3': 1, '4': 3, '5': 11, '6': '.api.WifiState.Interface', '10': 'interfaces'},
  ],
  '3': const [WifiState_AccessPoint$json, WifiState_Interface$json],
};

@$core.Deprecated('Use wifiStateDescriptor instead')
const WifiState_AccessPoint$json = const {
  '1': 'AccessPoint',
  '2': const [
    const {'1': 'ssid', '3': 1, '4': 1, '5': 9, '10': 'ssid'},
    const {'1': 'bssid', '3': 2, '4': 1, '5': 9, '10': 'bssid'},
    const {'1': 'signal_strength', '3': 3, '4': 1, '5': 1, '10': 'signalStrength'},
    const {'1': 'security', '3': 4, '4': 1, '5': 9, '10': 'security'},
  ],
};

@$core.Deprecated('Use wifiStateDescriptor instead')
const WifiState_Interface$json = const {
  '1': 'Interface',
  '2': const [
    const {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    const {'1': 'type', '3': 2, '4': 1, '5': 14, '6': '.api.WifiState.Interface.Type', '10': 'type'},
    const {'1': 'connected_access_point_ssid', '3': 3, '4': 1, '5': 9, '10': 'connectedAccessPointSsid'},
    const {'1': 'discovered_access_points', '3': 4, '4': 3, '5': 11, '6': '.api.WifiState.AccessPoint', '10': 'discoveredAccessPoints'},
    const {'1': 'autoconnect', '3': 5, '4': 1, '5': 8, '10': 'autoconnect'},
  ],
  '4': const [WifiState_Interface_Type$json],
};

@$core.Deprecated('Use wifiStateDescriptor instead')
const WifiState_Interface_Type$json = const {
  '1': 'Type',
  '2': const [
    const {'1': 'UNKNOWN', '2': 0},
    const {'1': 'WIFI', '2': 1},
    const {'1': 'ETHERNET', '2': 2},
    const {'1': 'AP', '2': 3},
  ],
};

/// Descriptor for `WifiState`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List wifiStateDescriptor = $convert.base64Decode('CglXaWZpU3RhdGUSOAoKaW50ZXJmYWNlcxgBIAMoCzIYLmFwaS5XaWZpU3RhdGUuSW50ZXJmYWNlUgppbnRlcmZhY2VzGnwKC0FjY2Vzc1BvaW50EhIKBHNzaWQYASABKAlSBHNzaWQSFAoFYnNzaWQYAiABKAlSBWJzc2lkEicKD3NpZ25hbF9zdHJlbmd0aBgDIAEoAVIOc2lnbmFsU3RyZW5ndGgSGgoIc2VjdXJpdHkYBCABKAlSCHNlY3VyaXR5GroCCglJbnRlcmZhY2USDgoCaWQYASABKAlSAmlkEjEKBHR5cGUYAiABKA4yHS5hcGkuV2lmaVN0YXRlLkludGVyZmFjZS5UeXBlUgR0eXBlEj0KG2Nvbm5lY3RlZF9hY2Nlc3NfcG9pbnRfc3NpZBgDIAEoCVIYY29ubmVjdGVkQWNjZXNzUG9pbnRTc2lkElQKGGRpc2NvdmVyZWRfYWNjZXNzX3BvaW50cxgEIAMoCzIaLmFwaS5XaWZpU3RhdGUuQWNjZXNzUG9pbnRSFmRpc2NvdmVyZWRBY2Nlc3NQb2ludHMSIAoLYXV0b2Nvbm5lY3QYBSABKAhSC2F1dG9jb25uZWN0IjMKBFR5cGUSCwoHVU5LTk9XThAAEggKBFdJRkkQARIMCghFVEhFUk5FVBACEgYKAkFQEAM=');
@$core.Deprecated('Use videoInputDeviceDescriptor instead')
const VideoInputDevice$json = const {
  '1': 'VideoInputDevice',
  '2': const [
    const {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    const {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `VideoInputDevice`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List videoInputDeviceDescriptor = $convert.base64Decode('ChBWaWRlb0lucHV0RGV2aWNlEg4KAmlkGAEgASgJUgJpZBISCgRuYW1lGAIgASgJUgRuYW1l');
@$core.Deprecated('Use audioInputDeviceDescriptor instead')
const AudioInputDevice$json = const {
  '1': 'AudioInputDevice',
  '2': const [
    const {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    const {'1': 'name', '3': 2, '4': 1, '5': 9, '10': 'name'},
  ],
};

/// Descriptor for `AudioInputDevice`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List audioInputDeviceDescriptor = $convert.base64Decode('ChBBdWRpb0lucHV0RGV2aWNlEg4KAmlkGAEgASgJUgJpZBISCgRuYW1lGAIgASgJUgRuYW1l');
@$core.Deprecated('Use deviceStateDescriptor instead')
const DeviceState$json = const {
  '1': 'DeviceState',
  '2': const [
    const {'1': 'video_input_devices', '3': 1, '4': 3, '5': 11, '6': '.api.VideoInputDevice', '10': 'videoInputDevices'},
    const {'1': 'audio_input_devices', '3': 2, '4': 3, '5': 11, '6': '.api.AudioInputDevice', '10': 'audioInputDevices'},
  ],
};

/// Descriptor for `DeviceState`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deviceStateDescriptor = $convert.base64Decode('CgtEZXZpY2VTdGF0ZRJFChN2aWRlb19pbnB1dF9kZXZpY2VzGAEgAygLMhUuYXBpLlZpZGVvSW5wdXREZXZpY2VSEXZpZGVvSW5wdXREZXZpY2VzEkUKE2F1ZGlvX2lucHV0X2RldmljZXMYAiADKAsyFS5hcGkuQXVkaW9JbnB1dERldmljZVIRYXVkaW9JbnB1dERldmljZXM=');
