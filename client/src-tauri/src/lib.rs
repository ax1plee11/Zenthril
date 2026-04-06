use tauri::Manager;
use tauri_plugin_store::StoreExt;

const STORE_PATH: &str = "vibrora_secure.json";
const PRIVATE_KEY_KEY: &str = "private_key";

/// Сохраняет приватный ключ в защищённое хранилище Tauri Store.
#[tauri::command]
fn store_private_key(app: tauri::AppHandle, key: String) -> Result<(), String> {
    let store = app
        .store(STORE_PATH)
        .map_err(|e| format!("Failed to open store: {e}"))?;
    store
        .set(PRIVATE_KEY_KEY, serde_json::Value::String(key));
    store.save().map_err(|e| format!("Failed to save store: {e}"))
}

/// Загружает приватный ключ из защищённого хранилища.
#[tauri::command]
fn load_private_key(app: tauri::AppHandle) -> Result<Option<String>, String> {
    let store = app
        .store(STORE_PATH)
        .map_err(|e| format!("Failed to open store: {e}"))?;
    let value = store.get(PRIVATE_KEY_KEY);
    match value {
        Some(serde_json::Value::String(s)) => Ok(Some(s)),
        Some(_) => Ok(None),
        None => Ok(None),
    }
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
            show_notification,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
