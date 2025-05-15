package com.trackerr.provisioner;

import android.app.Activity;
import android.view.View;
import androidx.appcompat.app.AlertDialog;
import com.google.android.material.snackbar.Snackbar;


public class Utils {
    public static void blockUserInteractions(View view, boolean block) {
        if (block) {
            view.setEnabled(false);
            view.setAlpha(0.2f);
        } else {
            view.setEnabled(true);
            view.setAlpha(1f);
        }
    }

    public static void showMessage(View view, String message){
        Snackbar snack = Snackbar.make(view, message, Snackbar.LENGTH_LONG);
        snack.setDuration(5000);
        snack.show();
    }

    public static void showCriticalMsg(Activity activity, String title, String msg){
        new AlertDialog.Builder(activity)
                .setTitle(title)
                .setMessage(msg)
                .setCancelable(false)
                .setPositiveButton("Exit", (dialog, which) -> {
                    activity.finishAffinity();
                    System.exit(0);
                })
                .show();
    }
}
