///
//  Generated code. Do not modify.
//  source: muxer.proto
//
// @dart = 2.12
// ignore_for_file: annotate_overrides,camel_case_types,unnecessary_const,non_constant_identifier_names,library_prefixes,unused_import,unused_shown_name,return_of_invalid_type,unnecessary_this,prefer_final_fields

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import 'muxer.pbenum.dart';

export 'muxer.pbenum.dart';

class WifiConnectRequest extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'WifiConnectRequest', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'api'), createEmptyInstance: create)
    ..aOS(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'interfaceId')
    ..aOS(2, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'bssid')
    ..aOS(3, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'password')
    ..aOS(4, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'apModeSsid')
    ..aOB(5, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'autoconnect')
    ..hasRequiredFields = false
  ;

  WifiConnectRequest._() : super();
  factory WifiConnectRequest({
    $core.String? interfaceId,
    $core.String? bssid,
    $core.String? password,
    $core.String? apModeSsid,
    $core.bool? autoconnect,
  }) {
    final _result = create();
    if (interfaceId != null) {
      _result.interfaceId = interfaceId;
    }
    if (bssid != null) {
      _result.bssid = bssid;
    }
    if (password != null) {
      _result.password = password;
    }
    if (apModeSsid != null) {
      _result.apModeSsid = apModeSsid;
    }
    if (autoconnect != null) {
      _result.autoconnect = autoconnect;
    }
    return _result;
  }
  factory WifiConnectRequest.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory WifiConnectRequest.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  WifiConnectRequest clone() => WifiConnectRequest()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  WifiConnectRequest copyWith(void Function(WifiConnectRequest) updates) => super.copyWith((message) => updates(message as WifiConnectRequest)) as WifiConnectRequest; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static WifiConnectRequest create() => WifiConnectRequest._();
  WifiConnectRequest createEmptyInstance() => create();
  static $pb.PbList<WifiConnectRequest> createRepeated() => $pb.PbList<WifiConnectRequest>();
  @$core.pragma('dart2js:noInline')
  static WifiConnectRequest getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<WifiConnectRequest>(create);
  static WifiConnectRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get interfaceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set interfaceId($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasInterfaceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearInterfaceId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get bssid => $_getSZ(1);
  @$pb.TagNumber(2)
  set bssid($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasBssid() => $_has(1);
  @$pb.TagNumber(2)
  void clearBssid() => clearField(2);

  @$pb.TagNumber(3)
  $core.String get password => $_getSZ(2);
  @$pb.TagNumber(3)
  set password($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasPassword() => $_has(2);
  @$pb.TagNumber(3)
  void clearPassword() => clearField(3);

  @$pb.TagNumber(4)
  $core.String get apModeSsid => $_getSZ(3);
  @$pb.TagNumber(4)
  set apModeSsid($core.String v) { $_setString(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasApModeSsid() => $_has(3);
  @$pb.TagNumber(4)
  void clearApModeSsid() => clearField(4);

  @$pb.TagNumber(5)
  $core.bool get autoconnect => $_getBF(4);
  @$pb.TagNumber(5)
  set autoconnect($core.bool v) { $_setBool(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasAutoconnect() => $_has(4);
  @$pb.TagNumber(5)
  void clearAutoconnect() => clearField(5);
}

class WifiConnectResponse extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'WifiConnectResponse', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'api'), createEmptyInstance: create)
    ..aOS(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'error')
    ..aOM<WifiState>(2, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'wifiState', subBuilder: WifiState.create)
    ..hasRequiredFields = false
  ;

  WifiConnectResponse._() : super();
  factory WifiConnectResponse({
    $core.String? error,
    WifiState? wifiState,
  }) {
    final _result = create();
    if (error != null) {
      _result.error = error;
    }
    if (wifiState != null) {
      _result.wifiState = wifiState;
    }
    return _result;
  }
  factory WifiConnectResponse.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory WifiConnectResponse.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  WifiConnectResponse clone() => WifiConnectResponse()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  WifiConnectResponse copyWith(void Function(WifiConnectResponse) updates) => super.copyWith((message) => updates(message as WifiConnectResponse)) as WifiConnectResponse; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static WifiConnectResponse create() => WifiConnectResponse._();
  WifiConnectResponse createEmptyInstance() => create();
  static $pb.PbList<WifiConnectResponse> createRepeated() => $pb.PbList<WifiConnectResponse>();
  @$core.pragma('dart2js:noInline')
  static WifiConnectResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<WifiConnectResponse>(create);
  static WifiConnectResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get error => $_getSZ(0);
  @$pb.TagNumber(1)
  set error($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasError() => $_has(0);
  @$pb.TagNumber(1)
  void clearError() => clearField(1);

  @$pb.TagNumber(2)
  WifiState get wifiState => $_getN(1);
  @$pb.TagNumber(2)
  set wifiState(WifiState v) { setField(2, v); }
  @$pb.TagNumber(2)
  $core.bool hasWifiState() => $_has(1);
  @$pb.TagNumber(2)
  void clearWifiState() => clearField(2);
  @$pb.TagNumber(2)
  WifiState ensureWifiState() => $_ensure(1);
}

class WifiState_AccessPoint extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'WifiState.AccessPoint', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'api'), createEmptyInstance: create)
    ..aOS(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'ssid')
    ..aOS(2, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'bssid')
    ..a<$core.double>(3, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'signalStrength', $pb.PbFieldType.OD)
    ..aOS(4, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'security')
    ..hasRequiredFields = false
  ;

  WifiState_AccessPoint._() : super();
  factory WifiState_AccessPoint({
    $core.String? ssid,
    $core.String? bssid,
    $core.double? signalStrength,
    $core.String? security,
  }) {
    final _result = create();
    if (ssid != null) {
      _result.ssid = ssid;
    }
    if (bssid != null) {
      _result.bssid = bssid;
    }
    if (signalStrength != null) {
      _result.signalStrength = signalStrength;
    }
    if (security != null) {
      _result.security = security;
    }
    return _result;
  }
  factory WifiState_AccessPoint.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory WifiState_AccessPoint.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  WifiState_AccessPoint clone() => WifiState_AccessPoint()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  WifiState_AccessPoint copyWith(void Function(WifiState_AccessPoint) updates) => super.copyWith((message) => updates(message as WifiState_AccessPoint)) as WifiState_AccessPoint; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static WifiState_AccessPoint create() => WifiState_AccessPoint._();
  WifiState_AccessPoint createEmptyInstance() => create();
  static $pb.PbList<WifiState_AccessPoint> createRepeated() => $pb.PbList<WifiState_AccessPoint>();
  @$core.pragma('dart2js:noInline')
  static WifiState_AccessPoint getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<WifiState_AccessPoint>(create);
  static WifiState_AccessPoint? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get ssid => $_getSZ(0);
  @$pb.TagNumber(1)
  set ssid($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasSsid() => $_has(0);
  @$pb.TagNumber(1)
  void clearSsid() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get bssid => $_getSZ(1);
  @$pb.TagNumber(2)
  set bssid($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasBssid() => $_has(1);
  @$pb.TagNumber(2)
  void clearBssid() => clearField(2);

  @$pb.TagNumber(3)
  $core.double get signalStrength => $_getN(2);
  @$pb.TagNumber(3)
  set signalStrength($core.double v) { $_setDouble(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasSignalStrength() => $_has(2);
  @$pb.TagNumber(3)
  void clearSignalStrength() => clearField(3);

  @$pb.TagNumber(4)
  $core.String get security => $_getSZ(3);
  @$pb.TagNumber(4)
  set security($core.String v) { $_setString(3, v); }
  @$pb.TagNumber(4)
  $core.bool hasSecurity() => $_has(3);
  @$pb.TagNumber(4)
  void clearSecurity() => clearField(4);
}

class WifiState_Interface extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'WifiState.Interface', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'api'), createEmptyInstance: create)
    ..aOS(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'id')
    ..e<WifiState_Interface_Type>(2, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'type', $pb.PbFieldType.OE, defaultOrMaker: WifiState_Interface_Type.UNKNOWN, valueOf: WifiState_Interface_Type.valueOf, enumValues: WifiState_Interface_Type.values)
    ..aOS(3, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'connectedAccessPointSsid')
    ..pc<WifiState_AccessPoint>(4, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'discoveredAccessPoints', $pb.PbFieldType.PM, subBuilder: WifiState_AccessPoint.create)
    ..aOB(5, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'autoconnect')
    ..hasRequiredFields = false
  ;

  WifiState_Interface._() : super();
  factory WifiState_Interface({
    $core.String? id,
    WifiState_Interface_Type? type,
    $core.String? connectedAccessPointSsid,
    $core.Iterable<WifiState_AccessPoint>? discoveredAccessPoints,
    $core.bool? autoconnect,
  }) {
    final _result = create();
    if (id != null) {
      _result.id = id;
    }
    if (type != null) {
      _result.type = type;
    }
    if (connectedAccessPointSsid != null) {
      _result.connectedAccessPointSsid = connectedAccessPointSsid;
    }
    if (discoveredAccessPoints != null) {
      _result.discoveredAccessPoints.addAll(discoveredAccessPoints);
    }
    if (autoconnect != null) {
      _result.autoconnect = autoconnect;
    }
    return _result;
  }
  factory WifiState_Interface.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory WifiState_Interface.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  WifiState_Interface clone() => WifiState_Interface()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  WifiState_Interface copyWith(void Function(WifiState_Interface) updates) => super.copyWith((message) => updates(message as WifiState_Interface)) as WifiState_Interface; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static WifiState_Interface create() => WifiState_Interface._();
  WifiState_Interface createEmptyInstance() => create();
  static $pb.PbList<WifiState_Interface> createRepeated() => $pb.PbList<WifiState_Interface>();
  @$core.pragma('dart2js:noInline')
  static WifiState_Interface getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<WifiState_Interface>(create);
  static WifiState_Interface? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => clearField(1);

  @$pb.TagNumber(2)
  WifiState_Interface_Type get type => $_getN(1);
  @$pb.TagNumber(2)
  set type(WifiState_Interface_Type v) { setField(2, v); }
  @$pb.TagNumber(2)
  $core.bool hasType() => $_has(1);
  @$pb.TagNumber(2)
  void clearType() => clearField(2);

  @$pb.TagNumber(3)
  $core.String get connectedAccessPointSsid => $_getSZ(2);
  @$pb.TagNumber(3)
  set connectedAccessPointSsid($core.String v) { $_setString(2, v); }
  @$pb.TagNumber(3)
  $core.bool hasConnectedAccessPointSsid() => $_has(2);
  @$pb.TagNumber(3)
  void clearConnectedAccessPointSsid() => clearField(3);

  @$pb.TagNumber(4)
  $core.List<WifiState_AccessPoint> get discoveredAccessPoints => $_getList(3);

  @$pb.TagNumber(5)
  $core.bool get autoconnect => $_getBF(4);
  @$pb.TagNumber(5)
  set autoconnect($core.bool v) { $_setBool(4, v); }
  @$pb.TagNumber(5)
  $core.bool hasAutoconnect() => $_has(4);
  @$pb.TagNumber(5)
  void clearAutoconnect() => clearField(5);
}

class WifiState extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'WifiState', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'api'), createEmptyInstance: create)
    ..pc<WifiState_Interface>(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'interfaces', $pb.PbFieldType.PM, subBuilder: WifiState_Interface.create)
    ..hasRequiredFields = false
  ;

  WifiState._() : super();
  factory WifiState({
    $core.Iterable<WifiState_Interface>? interfaces,
  }) {
    final _result = create();
    if (interfaces != null) {
      _result.interfaces.addAll(interfaces);
    }
    return _result;
  }
  factory WifiState.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory WifiState.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  WifiState clone() => WifiState()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  WifiState copyWith(void Function(WifiState) updates) => super.copyWith((message) => updates(message as WifiState)) as WifiState; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static WifiState create() => WifiState._();
  WifiState createEmptyInstance() => create();
  static $pb.PbList<WifiState> createRepeated() => $pb.PbList<WifiState>();
  @$core.pragma('dart2js:noInline')
  static WifiState getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<WifiState>(create);
  static WifiState? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<WifiState_Interface> get interfaces => $_getList(0);
}

class VideoInputDevice extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'VideoInputDevice', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'api'), createEmptyInstance: create)
    ..aOS(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'id')
    ..aOS(2, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'name')
    ..hasRequiredFields = false
  ;

  VideoInputDevice._() : super();
  factory VideoInputDevice({
    $core.String? id,
    $core.String? name,
  }) {
    final _result = create();
    if (id != null) {
      _result.id = id;
    }
    if (name != null) {
      _result.name = name;
    }
    return _result;
  }
  factory VideoInputDevice.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory VideoInputDevice.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  VideoInputDevice clone() => VideoInputDevice()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  VideoInputDevice copyWith(void Function(VideoInputDevice) updates) => super.copyWith((message) => updates(message as VideoInputDevice)) as VideoInputDevice; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static VideoInputDevice create() => VideoInputDevice._();
  VideoInputDevice createEmptyInstance() => create();
  static $pb.PbList<VideoInputDevice> createRepeated() => $pb.PbList<VideoInputDevice>();
  @$core.pragma('dart2js:noInline')
  static VideoInputDevice getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<VideoInputDevice>(create);
  static VideoInputDevice? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => clearField(2);
}

class AudioInputDevice extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'AudioInputDevice', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'api'), createEmptyInstance: create)
    ..aOS(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'id')
    ..aOS(2, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'name')
    ..hasRequiredFields = false
  ;

  AudioInputDevice._() : super();
  factory AudioInputDevice({
    $core.String? id,
    $core.String? name,
  }) {
    final _result = create();
    if (id != null) {
      _result.id = id;
    }
    if (name != null) {
      _result.name = name;
    }
    return _result;
  }
  factory AudioInputDevice.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory AudioInputDevice.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  AudioInputDevice clone() => AudioInputDevice()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  AudioInputDevice copyWith(void Function(AudioInputDevice) updates) => super.copyWith((message) => updates(message as AudioInputDevice)) as AudioInputDevice; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static AudioInputDevice create() => AudioInputDevice._();
  AudioInputDevice createEmptyInstance() => create();
  static $pb.PbList<AudioInputDevice> createRepeated() => $pb.PbList<AudioInputDevice>();
  @$core.pragma('dart2js:noInline')
  static AudioInputDevice getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<AudioInputDevice>(create);
  static AudioInputDevice? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String v) { $_setString(0, v); }
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => clearField(1);

  @$pb.TagNumber(2)
  $core.String get name => $_getSZ(1);
  @$pb.TagNumber(2)
  set name($core.String v) { $_setString(1, v); }
  @$pb.TagNumber(2)
  $core.bool hasName() => $_has(1);
  @$pb.TagNumber(2)
  void clearName() => clearField(2);
}

class DeviceState extends $pb.GeneratedMessage {
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'DeviceState', package: const $pb.PackageName(const $core.bool.fromEnvironment('protobuf.omit_message_names') ? '' : 'api'), createEmptyInstance: create)
    ..pc<VideoInputDevice>(1, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'videoInputDevices', $pb.PbFieldType.PM, subBuilder: VideoInputDevice.create)
    ..pc<AudioInputDevice>(2, const $core.bool.fromEnvironment('protobuf.omit_field_names') ? '' : 'audioInputDevices', $pb.PbFieldType.PM, subBuilder: AudioInputDevice.create)
    ..hasRequiredFields = false
  ;

  DeviceState._() : super();
  factory DeviceState({
    $core.Iterable<VideoInputDevice>? videoInputDevices,
    $core.Iterable<AudioInputDevice>? audioInputDevices,
  }) {
    final _result = create();
    if (videoInputDevices != null) {
      _result.videoInputDevices.addAll(videoInputDevices);
    }
    if (audioInputDevices != null) {
      _result.audioInputDevices.addAll(audioInputDevices);
    }
    return _result;
  }
  factory DeviceState.fromBuffer($core.List<$core.int> i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(i, r);
  factory DeviceState.fromJson($core.String i, [$pb.ExtensionRegistry r = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(i, r);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.deepCopy] instead. '
  'Will be removed in next major version')
  DeviceState clone() => DeviceState()..mergeFromMessage(this);
  @$core.Deprecated(
  'Using this can add significant overhead to your binary. '
  'Use [GeneratedMessageGenericExtensions.rebuild] instead. '
  'Will be removed in next major version')
  DeviceState copyWith(void Function(DeviceState) updates) => super.copyWith((message) => updates(message as DeviceState)) as DeviceState; // ignore: deprecated_member_use
  $pb.BuilderInfo get info_ => _i;
  @$core.pragma('dart2js:noInline')
  static DeviceState create() => DeviceState._();
  DeviceState createEmptyInstance() => create();
  static $pb.PbList<DeviceState> createRepeated() => $pb.PbList<DeviceState>();
  @$core.pragma('dart2js:noInline')
  static DeviceState getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<DeviceState>(create);
  static DeviceState? _defaultInstance;

  @$pb.TagNumber(1)
  $core.List<VideoInputDevice> get videoInputDevices => $_getList(0);

  @$pb.TagNumber(2)
  $core.List<AudioInputDevice> get audioInputDevices => $_getList(1);
}

