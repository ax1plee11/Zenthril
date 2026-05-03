import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

import en from './locales/en.json';
import ru from './locales/ru.json';
import uk from './locales/uk.json';

// Функция для определения языка по IP (можно интегрировать с API геолокации)
const detectLanguageByLocation = async (): Promise<string | null> => {
  try {
    // Можно использовать API типа ipapi.co или ip-api.com
    // Пример: const response = await fetch('https://ipapi.co/json/');
    // const data = await response.json();
    // const countryToLanguage: Record<string, string> = {
    //   'RU': 'ru', 'BY': 'ru', 'KZ': 'ru',
    //   'UA': 'uk',
    //   'US': 'en', 'GB': 'en', 'CA': 'en', 'AU': 'en',
    // };
    // return countryToLanguage[data.country_code] || 'en';
    
    // Пока используем язык браузера
    return null;
  } catch (error) {
    console.error('Failed to detect language by location:', error);
    return null;
  }
};

// Инициализация i18n
i18n
  .use(LanguageDetector) // Автоматическое определение языка
  .use(initReactI18next) // Интеграция с React
  .init({
    resources: {
      en: { translation: en },
      ru: { translation: ru },
      uk: { translation: uk },
    },
    fallbackLng: 'en', // Язык по умолчанию
    supportedLngs: ['en', 'ru', 'uk'],
    
    // Настройки определения языка
    detection: {
      order: [
        'localStorage',      // Сначала проверяем сохранённый выбор
        'navigator',         // Затем язык браузера
        'htmlTag',          // Затем HTML lang атрибут
      ],
      caches: ['localStorage'], // Сохраняем выбор в localStorage
      lookupLocalStorage: 'i18nextLng',
    },

    interpolation: {
      escapeValue: false, // React уже защищает от XSS
    },

    react: {
      useSuspense: false,
    },
  });

// Попытка определить язык по геолокации при первом запуске
if (!localStorage.getItem('i18nextLng')) {
  detectLanguageByLocation().then((lang) => {
    if (lang) {
      i18n.changeLanguage(lang);
    }
  });
}

export default i18n;

// Экспорт типов для TypeScript
export type Language = 'en' | 'ru' | 'uk';

export const languages: { code: Language; name: string; flag: string }[] = [
  { code: 'en', name: 'English', flag: '🇬🇧' },
  { code: 'ru', name: 'Русский', flag: '🇷🇺' },
  { code: 'uk', name: 'Українська', flag: '🇺🇦' },
];
