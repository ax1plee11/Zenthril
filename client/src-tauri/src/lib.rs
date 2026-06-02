use tauri::Manager;
use tauri_plugin_store::StoreExt;

const STORE_PATH: &str = "zenthril_secure.json";
const KEYRING_SERVICE: &str = "zenthril";
const PRIVATE_KEY_KEY: &str = "private_key";
const DEVICE_KEY_PREFIX: &str = "device_key_bundle:";
const PRIVATE_KEY_ACCOUNT: &str = "private_key";
const DEVICE_KEY_ACCOUNT_PREFIX: &str = "device_key_bundle:";

fn keyring_entry(account: &str) -> Result<keyring::Entry, String> {
    keyring::Entry::new(KEYRING_SERVICE, account)
        .map_err(|e| format!("Failed to open OS keychain entry: {e}"))
}

// SECURITY: OS keychain is the preferred desktop storage backend for private
// E2EE material. Legacy Tauri Store is kept only as a migration fallback.
fn keyring_set(account: &str, value: &str) -> Result<(), String> {
    keyring_entry(account)?
        .set_password(value)
        .map_err(|e| format!("Failed to save secret in OS keychain: {e}"))
}

fn keyring_get(account: &str) -> Result<Option<String>, String> {
    match keyring_entry(account)?.get_password() {
        Ok(value) => Ok(Some(value)),
        Err(keyring::Error::NoEntry) => Ok(None),
        Err(e) => Err(format!("Failed to load secret from OS keychain: {e}")),
    }
}

fn keyring_delete(account: &str) -> Result<(), String> {
    match keyring_entry(account)?.delete_credential() {
        Ok(()) | Err(keyring::Error::NoEntry) => Ok(()),
        Err(e) => Err(format!("Failed to delete secret from OS keychain: {e}")),
    }
}

fn legacy_store_get(app: &tauri::AppHandle, key: String) -> Result<Option<String>, String> {
    let store = app
        .store(STORE_PATH)
        .map_err(|e| format!("Failed to open legacy store: {e}"))?;
    match store.get(key) {
        Some(serde_json::Value::String(s)) => Ok(Some(s)),
        Some(_) => Ok(None),
        None => Ok(None),
    }
}

fn legacy_store_delete(app: &tauri::AppHandle, key: String) -> Result<(), String> {
    let store = app
        .store(STORE_PATH)
        .map_err(|e| format!("Failed to open legacy store: {e}"))?;
    store.delete(key);
    store
        .save()
        .map_err(|e| format!("Failed to save legacy store: {e}"))
}

/// Stores the legacy private key in the OS keychain.
#[tauri::command]
// SECURITY: this command writes to OS-backed key storage. The legacy Tauri
// Store path is read only during migration and then deleted.
fn store_private_key(app: tauri::AppHandle, key: String) -> Result<(), String> {
	let _ = app;
	keyring_set(PRIVATE_KEY_ACCOUNT, &key)
}

/// Loads the legacy private key from OS keychain, migrating from Tauri Store if needed.
#[tauri::command]
// SECURITY: migration removes the legacy Tauri Store copy after a successful
// keychain write, so private material does not remain duplicated.
fn load_private_key(app: tauri::AppHandle) -> Result<Option<String>, String> {
    if let Some(value) = keyring_get(PRIVATE_KEY_ACCOUNT)? {
        return Ok(Some(value));
    }
    let legacy = legacy_store_get(&app, PRIVATE_KEY_KEY.to_string())?;
    if let Some(value) = legacy {
        keyring_set(PRIVATE_KEY_ACCOUNT, &value)?;
        legacy_store_delete(&app, PRIVATE_KEY_KEY.to_string())?;
        return Ok(Some(value));
    }
    Ok(None)
}

#[tauri::command]
fn delete_private_key(app: tauri::AppHandle) -> Result<(), String> {
    keyring_delete(PRIVATE_KEY_ACCOUNT)?;
    legacy_store_delete(&app, PRIVATE_KEY_KEY.to_string())
}

#[tauri::command]
fn store_device_key_bundle(
	app: tauri::AppHandle,
	user_id: String,
	bundle_json: String,
) -> Result<(), String> {
	let _ = app;
	keyring_set(&format!("{DEVICE_KEY_ACCOUNT_PREFIX}{user_id}"), &bundle_json)
}

#[tauri::command]
fn load_device_key_bundle(
    app: tauri::AppHandle,
    user_id: String,
) -> Result<Option<String>, String> {
    let account = format!("{DEVICE_KEY_ACCOUNT_PREFIX}{user_id}");
    if let Some(value) = keyring_get(&account)? {
        return Ok(Some(value));
    }

    let legacy_key = format!("{DEVICE_KEY_PREFIX}{user_id}");
    let legacy = legacy_store_get(&app, legacy_key.clone())?;
    if let Some(value) = legacy {
        keyring_set(&account, &value)?;
        legacy_store_delete(&app, legacy_key)?;
        return Ok(Some(value));
    }
    Ok(None)
}

#[tauri::command]
fn delete_device_key_bundle(app: tauri::AppHandle, user_id: String) -> Result<(), String> {
    keyring_delete(&format!("{DEVICE_KEY_ACCOUNT_PREFIX}{user_id}"))?;
    legacy_store_delete(&app, format!("{DEVICE_KEY_PREFIX}{user_id}"))
}

/// Показывает системное уведомление через tauri-plugin-notification.
#[tauri::command]
fn show_notification(
    app: tauri::AppHandle,
    title: String,
    body: String,
) -> Result<(), String> {
    use tauri_plugin_notification::NotificationExt;
    app.notification()
        .builder()
        .title(&title)
        .body(&body)
        .show()
        .map_err(|e| format!("Failed to show notification: {e}"))
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_store::Builder::default().build())
        .setup(|app| {
            #[cfg(debug_assertions)]
            {
                let window = app.get_webview_window("main").unwrap();
                window.open_devtools();
            }
            Ok(())
		})
		.invoke_handler(tauri::generate_handler![
			store_private_key,
			load_private_key,
			delete_private_key,
			store_device_key_bundle,
			load_device_key_bundle,
			delete_device_key_bundle,
			show_notification,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
