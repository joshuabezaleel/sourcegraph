export enum FilterType {
    repo = 'repo',
    repogroup = 'repogroup',
    repohasfile = 'repohasfile',
    repohascommitafter = 'repohascommitafter',
    file = 'file',
    type = 'type',
    case = 'case',
    lang = 'lang',
    fork = 'fork',
    archived = 'archived',
    visibility = 'visibility',
    count = 'count',
    timeout = 'timeout',
    before = 'before',
    after = 'after',
    author = 'author',
    committer = 'committer',
    message = 'message',
    content = 'content',
    patterntype = 'patterntype',
    index = 'index',
    stable = 'stable',
    // eslint-disable-next-line unicorn/prevent-abbreviations
    rev = 'rev',
}

/* eslint-disable unicorn/prevent-abbreviations */
export enum AliasedFilterType {
    r = 'repo',
    g = 'repogroup',
    f = 'file',
    l = 'lang',
    language = 'lang',
    until = 'before',
    since = 'after',
    m = 'message',
    msg = 'message',
    revision = 'rev',
}
/* eslint-enable unicorn/prevent-abbreviations */

export const isFilterType = (filter: string): filter is FilterType => filter in FilterType
export const isAliasedFilterType = (filter: string): boolean => filter in AliasedFilterType

export const filterTypeKeys: FilterType[] = Object.keys(FilterType) as FilterType[]
export const filterTypeKeysWithAliases: (FilterType | AliasedFilterType)[] = [
    ...filterTypeKeys,
    ...Object.keys(AliasedFilterType),
] as (FilterType | AliasedFilterType)[]

export enum NegatedFilters {
    repo = '-repo',
    file = '-file',
    lang = '-lang',
    r = '-r',
    f = '-f',
    l = '-l',
    repohasfile = '-repohasfile',
    content = '-content',
    committer = '-committer',
    author = '-author',
    message = '-message',
}

/** The list of filters that are able to be negated. */
export type NegatableFilter =
    | FilterType.repo
    | FilterType.file
    | FilterType.repohasfile
    | FilterType.lang
    | FilterType.content
    | FilterType.committer
    | FilterType.author
    | FilterType.message

export const isNegatableFilter = (filter: FilterType): filter is NegatableFilter =>
    Object.keys(NegatedFilters).includes(filter)

/** The list of all negated filters. i.e. all valid filters that have `-` as a suffix. */
export const negatedFilters = Object.values(NegatedFilters)

export const isNegatedFilter = (filter: string): filter is NegatedFilters =>
    negatedFilters.includes(filter as NegatedFilters)

const negatedFilterToNegatableFilter: { [key: string]: NegatableFilter } = {
    '-repo': FilterType.repo,
    '-file': FilterType.file,
    '-lang': FilterType.lang,
    '-r': FilterType.repo,
    '-f': FilterType.file,
    '-l': FilterType.lang,
    '-repohasfile': FilterType.repohasfile,
    '-content': FilterType.content,
    '-committer': FilterType.committer,
    '-author': FilterType.author,
    '-message': FilterType.message,
}

export const resolveNegatedFilter = (filter: NegatedFilters): NegatableFilter => negatedFilterToNegatableFilter[filter]
