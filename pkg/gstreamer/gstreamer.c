#include "gstreaner.h"

#include <gst/app/gstappsrc.h>
#include <stdio.h>

void gstreamer_init(void)
{
    gst_init(NULL, NULL);
}

void gstreamer_main_loop(void)
{
    g_main_loop_run(g_main_loop_new(NULL, FALSE));
}

static gboolean gstreamer_bus_call(GstBus *bus, GstMessage *msg, gpointer data)
{
    switch (GST_MESSAGE_TYPE(msg))
    {

    case GST_MESSAGE_EOS:
        g_print("End of stream\n");
        exit(1);
        break;

    case GST_MESSAGE_ERROR:
    {
        gchar *debug;
        GError *error;

        gst_message_parse_error(msg, &error, &debug);
        g_free(debug);

        g_printerr("Error: %s\n", error->message);
        g_error_free(error);
        exit(1);
    }
    default:
        break;
    }

    return TRUE;
}

static GstFlowReturn gstreamer_pull_rtp_buffer(GstElement *object, gpointer user_data)
{
    GstSample *sample = NULL;
    GstBuffer *buffer = NULL;
    gpointer copy = NULL;
    gsize copy_size = 0;

    g_signal_emit_by_name(object, "pull-sample", &sample);
    if (sample)
    {
        buffer = gst_sample_get_buffer(sample);
        if (buffer)
        {
            gst_buffer_extract_dup(buffer, 0, gst_buffer_get_size(buffer), &copy, &copy_size);
            goHandleRtpAppSinkBuffer(copy, copy_size, GST_BUFFER_DURATION(buffer), (void *)user_data);
        }
        gst_sample_unref(sample);
    }

    return GST_FLOW_OK;
}

static void gstreamer_twcc_stats(GObject *object, GParamSpec *spec, gpointer data)
{
    GstStructure *stats;
    g_object_get(object, "twcc-stats", &stats, NULL);

    guint bitrateSent, bitrateRecv, packetsSent, packetsRecv;
    gint64 avgDeltaOfDelta;

    gst_structure_get(stats,
                      "bitrate-sent", G_TYPE_UINT, &bitrateSent,
                      "bitrate-recv", G_TYPE_UINT, &bitrateRecv,
                      "packets-sent", G_TYPE_UINT, &packetsSent,
                      "packets-recv", G_TYPE_UINT, &packetsRecv,
                      "avg-delta-of-delta", G_TYPE_INT64, &avgDeltaOfDelta,
                      NULL);

    goHandleTwccStats(
        bitrateSent, bitrateRecv, packetsSent, packetsRecv, avgDeltaOfDelta, (void *)data);
    gst_structure_free(stats);
}

static GstCaps *gstreamer_request_pt_map(GstElement *rtpbin, guint session, guint pt, gpointer user_data)
{
    if (pt == 96)
    {
        return gst_caps_new_simple("application/x-rtp",
                                   "payload", G_TYPE_INT, 96,
                                   "media", G_TYPE_STRING, "video",
                                   "clock-rate", G_TYPE_INT, 90000,
                                   "encoding-name", G_TYPE_STRING, "VP8",
                                   "extmap-5", G_TYPE_STRING, "http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01",
                                   NULL);
    }
    else if (pt == 98)
    {
        return gst_caps_new_simple("application/x-rtp",
                                   "payload", G_TYPE_INT, 98,
                                   "media", G_TYPE_STRING, "video",
                                   "clock-rate", G_TYPE_INT, 90000,
                                   "encoding-name", G_TYPE_STRING, "VP9",
                                   NULL);
    }
    else if (pt == 111)
    {

        return gst_caps_new_simple("application/x-rtp",
                                   "payload", G_TYPE_INT, 111,
                                   "media", G_TYPE_STRING, "audio",
                                   "clock-rate", G_TYPE_INT, 48000,
                                   "encoding-name", G_TYPE_STRING, "OPUS",
                                   NULL);
    }
    else
    {
        return NULL;
    }
}

static GstElement *gstreamer_request_aux_sender(GstElement *rtpbin, guint sessid, gpointer user_data)
{
    GstElement *rtx, *bin;
    GstPad *pad;
    gchar *name;
    GstStructure *pt_map;

    bin = gst_bin_new(NULL);
    rtx = gst_element_factory_make("rtprtxsend", NULL);
    pt_map = gst_structure_new(
        "application/x-rtp-pt-map",
        "96", G_TYPE_UINT, 97,
        "111", G_TYPE_UINT, 112,
        NULL);
    g_object_set(rtx, "payload-type-map", pt_map, NULL);
    gst_structure_free(pt_map);
    gst_bin_add(GST_BIN(bin), rtx);

    pad = gst_element_get_static_pad(rtx, "src");
    name = g_strdup_printf("src_%u", sessid);
    gst_element_add_pad(bin, gst_ghost_pad_new(name, pad));
    g_free(name);
    gst_object_unref(pad);

    pad = gst_element_get_static_pad(rtx, "sink");
    name = g_strdup_printf("sink_%u", sessid);
    gst_element_add_pad(bin, gst_ghost_pad_new(name, pad));
    g_free(name);
    gst_object_unref(pad);

    return bin;
}

GstElement *gstreamer_start(char *pipelineStr, void *data)
{
    GstElement *pipeline = gst_parse_launch(pipelineStr, NULL);

    GstBus *bus = gst_pipeline_get_bus(GST_PIPELINE(pipeline));
    gst_bus_add_watch(bus, gstreamer_bus_call, NULL);
    gst_object_unref(bus);

    GstElement *rtpsink0 = gst_bin_get_by_name(GST_BIN(pipeline), "rtpsink");
    g_object_set(rtpsink0, "emit-signals", TRUE, NULL);
    g_signal_connect(rtpsink0, "new-sample", G_CALLBACK(gstreamer_pull_rtp_buffer), data);
    gst_object_unref(rtpsink0);

    GstElement *rtcpsink0 = gst_bin_get_by_name(GST_BIN(pipeline), "rtcpsink");
    g_object_set(rtcpsink0, "emit-signals", TRUE, NULL);
    g_signal_connect(rtcpsink0, "new-sample", G_CALLBACK(gstreamer_pull_rtp_buffer), data);
    gst_object_unref(rtcpsink0);

    GstElement *rtpbin = gst_bin_get_by_name(GST_BIN(pipeline), "rtpbin");
    g_signal_connect(rtpbin, "request-aux-sender", G_CALLBACK(gstreamer_request_aux_sender), data);
    g_signal_connect(rtpbin, "notify::twcc-stats", G_CALLBACK(gstreamer_twcc_stats), data);
    g_signal_connect(rtpbin, "request-pt-map", G_CALLBACK(gstreamer_request_pt_map), data);
    gst_object_unref(rtpbin);

    gst_element_set_state(pipeline, GST_STATE_PLAYING);

    return pipeline;
}

void gstreamer_set_bitrate(GstElement *pipeline, unsigned int bitrate)
{
    GstElement *encoder = gst_bin_get_by_name(GST_BIN(pipeline), "videoencoder");
    g_object_set(G_OBJECT(encoder), "target-bitrate", bitrate, NULL);
    gst_object_unref(encoder);
}

void gstreamer_push_rtcp(GstElement *pipeline, void *buffer, int len)
{
    GstElement *rtcpsrc0 = gst_bin_get_by_name(GST_BIN(pipeline), "rtcpsrc0");
    gpointer p0 = g_memdup(buffer, len);
    GstBuffer *wrapped0 = gst_buffer_new_wrapped(p0, len);
    gst_app_src_push_buffer(GST_APP_SRC(rtcpsrc0), wrapped0);
    gst_object_unref(rtcpsrc0);

    GstElement *rtcpsrc1 = gst_bin_get_by_name(GST_BIN(pipeline), "rtcpsrc1");
    gpointer p1 = g_memdup(buffer, len);
    GstBuffer *wrapped1 = gst_buffer_new_wrapped(p1, len);
    gst_app_src_push_buffer(GST_APP_SRC(rtcpsrc1), wrapped1);
    gst_object_unref(rtcpsrc1);
}