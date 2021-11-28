#include "gstreamer.h"

#include <gst/app/gstappsrc.h>
#include <stdio.h>

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

static GstFlowReturn gstreamer_pull_video_buffer(GstElement *object, gpointer user_data)
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
            goHandleVideoSinkBuffer(copy, copy_size, GST_BUFFER_DURATION(buffer), (void *)user_data);
        }
        gst_sample_unref(sample);
    }

    return GST_FLOW_OK;
}

static GstFlowReturn gstreamer_pull_audio_buffer(GstElement *object, gpointer user_data)
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
            goHandleAudioSinkBuffer(copy, copy_size, GST_BUFFER_DURATION(buffer), (void *)user_data);
        }
        gst_sample_unref(sample);
    }

    return GST_FLOW_OK;
}

GstElement *gstreamer_start(char *pipelineStr, void *data)
{
    GstElement *pipeline = gst_parse_launch(pipelineStr, NULL);

    GstElement *audio_sink = gst_bin_get_by_name(GST_BIN(pipeline), "audio_sink");
    g_object_set(audio_sink, "emit-signals", TRUE, NULL);
    g_signal_connect(audio_sink, "new-sample", G_CALLBACK(gstreamer_pull_rtp_buffer), data);
    gst_object_unref(audio_sink);

    GstElement *video_sink = gst_bin_get_by_name(GST_BIN(pipeline), "video_sink");
    g_object_set(video_sink, "emit-signals", TRUE, NULL);
    g_signal_connect(video_sink, "new-sample", G_CALLBACK(gstreamer_pull_rtp_buffer), data);
    gst_object_unref(video_sink);

    // GstElement *pipeline = gst_pipeline_new("pipeline");

    // GstElement *playbin = gst_element_factory_make("playbin", "source");
    // g_object_set(playbin, "uri", uri, NULL);

    // // create an audio pipeline
    // GstElement *audio_bin = gst_bin_new("audio_sink_bin");
    // GstElement *audio_convert = gst_element_factory_make("audioconvert", "audio_convert");
    // GstElement *audio_encode = gst_element_factory_make("opusenc", "audio_encode");
    // g_object_set(audio_encode,
    //              "inband-fec", TRUE,
    //              "packet-loss-percentage", 8, NULL);

    // // link appsink
    // GstElement *audio_packetize = gst_element_factory_make("rtpopuspay", "audio_packetize");
    // g_object_set(audio_packetize, "pt", 111, NULL);
    // GstElement *audio_sink = gst_element_factory_make("appsink", "audio_sink");
    // gst_bin_add_many(GST_BIN(audio_bin), audio_convert, audio_encode, audio_packetize, audio_sink, NULL);
    // gst_element_link_many(audio_convert, audio_encode, audio_packetize, audio_sink, NULL);
    // g_object_set(audio_sink, "emit-signals", TRUE, NULL);
    // g_signal_connect(audio_sink, "new-sample", G_CALLBACK(gstreamer_pull_rtp_buffer), data);

    // // create a video pipeline
    // GstElement *video_bin = gst_bin_new("video_sink_bin");
    // GstElement *video_queue = gst_element_factory_make("queue", "video_queue");
    // GstElement *video_convert = gst_element_factory_make("videoconvert", "video_convert");
    // GstElement *video_encode_in = gst_element_factory_make("capsfilter", "video_encode_in");
    // GstCaps *video_encode_incaps = gst_caps_new_simple("video/x-raw", "format", G_TYPE_STRING, "I420", NULL);
    // g_object_set(video_encode_in, "caps", video_encode_incaps, NULL);
    // GstElement *video_encode = gst_element_factory_make("vp8enc", "video_encode");
    // g_object_set(video_encode,
    //              "error-resilient", 2,
    //              "keyframe-max-dist", 10,
    //              "auto-alt-ref", TRUE,
    //              "cpu-used", 5,
    //              "deadline", 1, NULL);
    // GstElement *video_encode_out = gst_element_factory_make("capsfilter", "video_encode_out");
    // GstCaps *video_encode_outcaps = gst_caps_new_simple("video/x-vp8", "profile", G_TYPE_STRING, "0", NULL);
    // g_object_set(video_encode_out, "caps", video_encode_outcaps, NULL);
    // // link appsink
    // GstElement *video_packetize = gst_element_factory_make("rtpvp8pay", "video_packetize");
    // g_object_set(video_packetize, "pt", 96, NULL);
    // GstElement *video_sink = gst_element_factory_make("appsink", "video_sink");
    // gst_bin_add_many(GST_BIN(video_bin), video_convert, video_encode_in, video_encode, video_encode_out, video_packetize, video_sink, NULL);
    // gst_element_link_many(video_convert, video_encode_in, video_encode, video_encode_out, video_packetize, video_sink, NULL);
    // g_object_set(video_sink, "emit-signals", TRUE, NULL);
    // g_signal_connect(video_sink, "new-sample", G_CALLBACK(gstreamer_pull_rtp_buffer), data);

    // // link audio pads
    // GstPad *audio_pad = gst_element_get_static_pad(audio_convert, "sink");
    // GstPad *audio_ghost_pad = gst_ghost_pad_new("sink", audio_pad);
    // gst_pad_set_active(audio_ghost_pad, TRUE);
    // gst_element_add_pad(audio_bin, audio_ghost_pad);
    // gst_object_unref(audio_pad);

    // // link video pads
    // GstPad *video_pad = gst_element_get_static_pad(video_convert, "sink");
    // GstPad *video_ghost_pad = gst_ghost_pad_new("sink", video_pad);
    // gst_pad_set_active(video_ghost_pad, TRUE);
    // gst_element_add_pad(video_bin, video_ghost_pad);
    // gst_object_unref(video_pad);

    // // set the playbink sinks
    // g_object_set(GST_OBJECT(playbin), "audio-sink", audio_bin, NULL);
    // g_object_set(GST_OBJECT(playbin), "video-sink", video_bin, NULL);

    // // link to pipeline
    // gst_bin_add_many(GST_BIN(pipeline), playbin, NULL);

    gst_element_set_state(pipeline, GST_STATE_PLAYING);

    return pipeline;
}

void gstreamer_stop(GstElement *pipeline)
{
    gst_element_set_state(pipeline, GST_STATE_NULL);
    gst_object_unref(pipeline);
}

void gstreamer_set_video_bitrate(GstElement *pipeline, unsigned int bitrate)
{
    GstElement *encoder = gst_bin_get_by_name(GST_BIN(pipeline), "video_encode");
    g_object_set(G_OBJECT(encoder), "target-bitrate", bitrate, NULL);
    gst_object_unref(encoder);
}
