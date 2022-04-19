///
//  Generated code. Do not modify.
//  source: muxer.proto
//
// @dart = 2.12
// ignore_for_file: annotate_overrides,camel_case_types,unnecessary_const,non_constant_identifier_names,library_prefixes,unused_import,unused_shown_name,return_of_invalid_type,unnecessary_this,prefer_final_fields

// ignore_for_file: UNDEFINED_SHOWN_NAME
import 'dart:core' as $core;
import 'package:protobuf/protobuf.dart' as $pb;

class WifiState_Interface_Type extends $pb.ProtobufEnum {
  static const WifiState_Interface_Type UNKNOWN = WifiState_Interface_Type._(0, const $core.bool.fromEnvironment('protobuf.omit_enum_names') ? '' : 'UNKNOWN');
  static const WifiState_Interface_Type WIFI = WifiState_Interface_Type._(1, const $core.bool.fromEnvironment('protobuf.omit_enum_names') ? '' : 'WIFI');
  static const WifiState_Interface_Type ETHERNET = WifiState_Interface_Type._(2, const $core.bool.fromEnvironment('protobuf.omit_enum_names') ? '' : 'ETHERNET');
  static const WifiState_Interface_Type AP = WifiState_Interface_Type._(3, const $core.bool.fromEnvironment('protobuf.omit_enum_names') ? '' : 'AP');

  static const $core.List<WifiState_Interface_Type> values = <WifiState_Interface_Type> [
    UNKNOWN,
    WIFI,
    ETHERNET,
    AP,
  ];

  static final $core.Map<$core.int, WifiState_Interface_Type> _byValue = $pb.ProtobufEnum.initByValue(values);
  static WifiState_Interface_Type? valueOf($core.int value) => _byValue[value];

  const WifiState_Interface_Type._($core.int v, $core.String n) : super(v, n);
}

