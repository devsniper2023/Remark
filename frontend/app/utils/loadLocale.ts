/** this is generated file by "npm run generate-langs" **/
// it is ok that is empty. Default messages from code will be used.
const enMessages = {};

export async function loadLocale(locale: string): Promise<Record<string, string>> {
  if (locale === 'ru') {
    return import(/* webpackChunkName: "ru" */ '../locales/ru.json')
      .then(res => res.default)
      .catch(() => enMessages);
  }
  if (locale === 'de') {
    return import(/* webpackChunkName: "de" */ '../locales/de.json')
      .then(res => res.default)
      .catch(() => enMessages);
  }
  if (locale === 'fi') {
    return import(/* webpackChunkName: "fi" */ '../locales/fi.json')
      .then(res => res.default)
      .catch(() => enMessages);
  }

  return enMessages;
}
