import { filtersApi } from '~/api/filters';

// ログイン中のユーザーの保存済みフィルター。左ナビ（お気に入り）と課題一覧で共有し、変えたら reload で読み直す
export const useSavedFilters = () => {
  const filters = useState('savedFilters', () => []);
  const reload = async () => {
    try {
      filters.value = await filtersApi.fetchAll();
    } catch (err) {
      console.error('Failed to fetch filters:', err);
    }
  };
  return { filters, reload };
};
