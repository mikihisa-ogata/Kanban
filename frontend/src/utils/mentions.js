// メンションは Markdown の本文に「@[表示名](user:ユーザー ID)」の形で書く（bff の domain/mention.go と同じ書式）。
// 表示名に空白があっても、誰へのメンションかが ID で決まる

// 本文の中のメンションに一致する正規表現（1 つ目のグループが表示名、2 つ目がユーザー ID）
export const mentionPattern = () => /@\[([^\]\n]+)\]\(user:(\d+)\)/g;

// ユーザーへのメンションの書式を返す。表示名の「]」は書式を壊すので除く
export const mentionOf = (user) => `@[${user.DisplayName.replace(/[\]\n]/g, '')}](user:${user.ID})`;

// メンションを「@表示名」にした本文を返す（カードの説明の抜粋など、Markdown として表示しない場所で使う）
export const plainMentions = (text) => (text ?? '').replace(mentionPattern(), '@$1');
