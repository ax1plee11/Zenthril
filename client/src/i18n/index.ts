import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

import en from './locales/en.json';
import ru from './locales/ru.json';
import uk from './locales/uk.json';

const detectLanguageByLocation = async (): Promise<string | null> => {
  try {
    // Future hook for optional location-based language selection.
    // Example: const response = await fetch('https://ipapi.co/json/');
    // const data = await response.json();
    // const countryToLanguage: Record<string, string> = {
    //   'RU': 'ru', 'BY': 'ru', 'KZ': 'ru',
    //   'UA': 'uk',
    //   'US': 'en', 'GB': 'en', 'CA': 'en', 'AU': 'en',
    // };
    // return countryToLanguage[data.country_code] || 'ru';

    // Keep automatic location detection disabled until its privacy model is reviewed.
    return null;
  } catch (error) {
    console.error('Failed to detect language by location:', error);
    return null;
  }
};

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      ru: { translation: ru },
      en: { translation: en },
      uk: { translation: uk },
    },
    fallbackLng: 'ru',
    supportedLngs: ['ru', 'en', 'uk'],

    detection: {
      // Preserve an explicit user choice before consulting browser preferences.
      order: [
        'localStorage',
        'navigator',
        'htmlTag',
      ],
      caches: ['localStorage'],
      lookupLocalStorage: 'i18nextLng',
    },

    interpolation: {
      escapeValue: false,
    },

    react: {
      useSuspense: false,
    },
  });

// Use Russian when the user has no saved language preference.
if (!localStorage.getItem('i18nextLng')) {
  detectLanguageByLocation().then((lang) => {
    if (lang) {
      i18n.changeLanguage(lang);
    }
  });
}

export default i18n;

export type Language = 'ru' | 'en' | 'uk';

export const languages: { code: Language; name: string; flag: string }[] = [
  { code: 'ru', name: '\u0420\u0443\u0441\u0441\u043a\u0438\u0439', flag: 'RU' },
  { code: 'en', name: 'English', flag: 'EN' },
  { code: 'uk', name: '\u0423\u043a\u0440\u0430\u0457\u043d\u0441\u044c\u043a\u0430', flag: 'UK' },
];
