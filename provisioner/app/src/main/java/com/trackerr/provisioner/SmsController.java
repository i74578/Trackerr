package com.trackerr.provisioner;

import static androidx.core.content.ContextCompat.registerReceiver;
import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.content.IntentFilter;
import android.os.Bundle;
import android.telephony.SmsManager;
import android.telephony.SmsMessage;
import androidx.core.content.ContextCompat;
import java.util.concurrent.CompletableFuture;

public class SmsController {
    Context context;
    String pn, id;
    String[] initCommands,successKeyword;
    private int currentResponseIndex = 0;
    CompletableFuture<Void> future;

    // Flag to indicate if future should wait for SMS response prior to completion
    boolean waitForResponses;

    public SmsController(Context context, String phonenumber, String id, ModelCommands commands,boolean waitForResponses) {
        this.context = context;
        this.pn = phonenumber;
        this.id = id;
        this.initCommands = commands.init_commands;
        this.successKeyword = commands.success_keyword;
        this.waitForResponses = waitForResponses;

        if (waitForResponses) {
            // Register SMS receiver
            IntentFilter filter = new IntentFilter("android.provider.Telephony.SMS_RECEIVED");
            registerReceiver(context, SmsReceiver, filter, ContextCompat.RECEIVER_EXPORTED);
        }
    }

    public CompletableFuture<Void> start() {
        // Run seperate thread to send sms messages
        Thread t = new Thread(new Runnable(){
            @Override
            public void run() {
                for (String initCommand : initCommands) {
                    sendSMS(initCommand);
                    System.out.println("Sent: " + initCommand);
                    // Introduce a delay of 2s, to prevent D21L from ignoring messages
                    try {
                        Thread.sleep(2000);
                    } catch (InterruptedException e) {
                        throw new RuntimeException(e);
                    }
                }
            }
        });
        t.start();

        // Return non completed future if wait for sms responses is enabled
        if (waitForResponses) {
            future = new CompletableFuture<>();
            return future;
        }
        // Else just send a completed future
        else {
            return CompletableFuture.completedFuture(null);
        }
    }

    public void sendSMS(String payload) {
        SmsManager smsManager = context.getSystemService(SmsManager.class);
        smsManager.sendTextMessage(this.pn,
                null,
                payload,
                null,
                null);
    }

    private final BroadcastReceiver SmsReceiver = new BroadcastReceiver() {
        @Override
        public void onReceive(Context context, Intent intent) {
            Bundle bundle = intent.getExtras();
            if (bundle != null) {
                Object[] pdus = (Object[]) bundle.get("pdus");
                if (pdus != null) {
                    for (Object pdu : pdus) {
                        String format = bundle.getString("format");
                        SmsMessage smsMessage = SmsMessage.createFromPdu((byte[]) pdu, format);
                        // Check if sms sender matches the tracker phone number
                        if (smsMessage.getOriginatingAddress() != null && smsMessage.getOriginatingAddress().endsWith(SmsController.this.pn)) {
                            String res = smsMessage.getMessageBody();
                            // Check if message contains correct success keyword
                            if (res.contains(successKeyword[currentResponseIndex])) {
                                // Unregister Receiver and complete future if received all responses
                                if (++currentResponseIndex >= initCommands.length) {
                                    unregister();
                                    future.complete(null);
                                }
                            }
                        }
                    }
                }
            }
        }
    };

    public void unregister(){
        context.unregisterReceiver(SmsReceiver);
    }
}
