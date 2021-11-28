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
    GstElementFactory *factory = gst_element_factory_find("nvvidconv");
    GstElement *encoder = gst_bin_get_by_name(GST_BIN(pipeline), "video_encode");
    // detect jetson nano with nvvidconv
    g_object_set(G_OBJECT(encoder), factory == NULL ? "target-bitrate" : "bitrate", bitrate, NULL);
    gst_object_unref(encoder);
    if (factory != NULL)
    {
        gst_object_unref(factory);
    }
}
