package com.trackerr.provisioner;

import android.content.Context;
import android.content.SharedPreferences;
import android.os.Bundle;
import android.widget.Button;
import android.widget.EditText;

import static com.trackerr.provisioner.Utils.blockUserInteractions;
import static com.trackerr.provisioner.Utils.showMessage;

import androidx.appcompat.app.AppCompatActivity;
import androidx.constraintlayout.widget.ConstraintLayout;

public class LoginActivity extends AppCompatActivity {
    EditText keyField;
    ConstraintLayout loginLayout;

    @Override
    protected void onCreate(Bundle savedInstanceState){
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_login);
        keyField = findViewById(R.id.apiKeyField);
        loginLayout = findViewById(R.id.loginLayout);
        Button submitkeybtn = findViewById(R.id.submitKeyBtn);
        submitkeybtn.setOnClickListener(v -> validateAndSaveKey());
    }

    private void validateAndSaveKey(){
        String apikey = keyField.getText().toString();
        if (apikey.isEmpty()){
            showMessage(loginLayout,"Please type in the API key or press the back button to abort this action");
            return;
        }
        blockUserInteractions(loginLayout,true);
        apiClient apiClient = new apiClient(this, apikey);
        apiClient.whoami()
                .thenAccept(ownerName -> {
                    SharedPreferences sharedPreferences = this.getSharedPreferences("DATA", Context.MODE_PRIVATE);
                    sharedPreferences.edit().putString("api-key",apikey).apply();
                    sharedPreferences.edit().putString("key-owner",ownerName).apply();
                    finish();
                })
                .exceptionally(ex -> {
                    showMessage(loginLayout,"The provided API key is invalid");
                    blockUserInteractions(loginLayout,false);
                    return null;});
    }
}
