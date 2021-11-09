#include "gstreamer.h"

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

GstElement *gstreamer_start(char *pipelineStr, void *data)
{
    GstElement *pipeline = gst_parse_launch(pipelineStr, NULL);

    GstBus *bus = gst_pipeline_get_bus(GST_PIPELINE(pipeline));
    gst_bus_add_watch(bus, gstreamer_bus_call, NULL);
    gst_object_unref(bus);

    GstElement *videosink = gst_bin_get_by_name(GST_BIN(pipeline), "videosink");
    g_object_set(videosink, "emit-signals", TRUE, NULL);
    g_signal_connect(videosink, "new-sample", G_CALLBACK(gstreamer_pull_rtp_buffer), data);
    gst_object_unref(videosink);

    GstElement *audiosink = gst_bin_get_by_name(GST_BIN(pipeline), "audiosink");
    g_object_set(audiosink, "emit-signals", TRUE, NULL);
    g_signal_connect(audiosink, "new-sample", G_CALLBACK(gstreamer_pull_rtp_buffer), data);
    gst_object_unref(audiosink);

    gst_element_set_state(pipeline, GST_STATE_PLAYING);

    return pipeline;
}

void gstreamer_stop(GstElement *pipeline)
{
    gst_element_set_state(pipeline, GST_STATE_NULL);
    gst_object_unref(pipeline);
}

void gstreamer_set_bitrate(GstElement *pipeline, unsigned int bitrate)
{
    GstElement *encoder = gst_bin_get_by_name(GST_BIN(pipeline), "videoencoder");
    g_object_set(G_OBJECT(encoder), "target-bitrate", bitrate, NULL);
    gst_object_unref(encoder);
}
