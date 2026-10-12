<template>
	<div
		class="projects-overview loader-container"
		:class="{'is-loading': showLoading}"
	>
		<XButton
			:to="{name: 'project.create'}"
			class="is-pulled-end"
			icon="plus"
		>
			{{ $t('project.create.header') }}
		</XButton>

		<div class="content">
			<h1>{{ $t('projectFinance.overview.title') }}</h1>
			<p class="has-text-grey">
				{{ $t('projectFinance.overview.description') }}
			</p>
		</div>

		<section class="projects-overview__stats">
			<div class="projects-overview__stat">
				<span class="projects-overview__stat-label">{{ $t('projectFinance.overview.activeProjects') }}</span>
				<span class="projects-overview__stat-value">{{ activeCount }}</span>
			</div>
			<div class="projects-overview__stat">
				<span class="projects-overview__stat-label">{{ $t('projectFinance.overview.archivedProjects') }}</span>
				<span class="projects-overview__stat-value">{{ archivedCount }}</span>
			</div>
			<div
				v-for="total in totals"
				:key="total.currency"
				class="projects-overview__stat projects-overview__stat--money"
			>
				<span class="projects-overview__stat-label">
					{{ total.currency || $t('projectFinance.noCurrency') }} · {{ $t('projectFinance.overview.inView', {count: total.projects}) }}
				</span>
				<span class="projects-overview__stat-value">{{ formatMoney(total.owed, total.currency) }}</span>
				<span class="projects-overview__stat-sub">
					{{ $t('projectFinance.overview.owedOf', {paid: formatMoney(total.paid, total.currency), fee: formatMoney(total.fee, total.currency)}) }}
				</span>
			</div>
		</section>

		<section class="projects-overview__filters">
			<FormInput
				v-model="search"
				type="search"
				class="projects-overview__search"
				:placeholder="$t('projectFinance.overview.searchPlaceholder')"
				:aria-label="$t('projectFinance.overview.searchPlaceholder')"
			/>
			<FormSelect
				v-model="clientFilter"
				:aria-label="$t('projectFinance.client')"
				:options="clientOptions"
			/>
			<FormSelect
				v-model.number="workerFilter"
				:aria-label="$t('projectFinance.overview.worker')"
				:options="workerOptions"
			/>
			<FormSelect
				v-model="statusFilter"
				:aria-label="$t('projectFinance.overview.status')"
				:options="statusOptions"
			/>
		</section>

		<p
			v-if="!isPending && rows.length === 0"
			class="has-text-centered has-text-grey is-italic projects-overview__empty"
		>
			{{ items.length === 0 ? $t('projectFinance.overview.empty') : $t('projectFinance.overview.noMatch') }}
		</p>

		<div
			v-else
			class="projects-overview__table-wrap"
		>
			<table class="table is-striped is-hoverable is-fullwidth projects-overview__table">
				<thead>
					<tr>
						<th
							v-for="column in columns"
							:key="column.field"
							:aria-sort="ariaSort(column.field)"
							:class="{'has-text-right': column.numeric}"
						>
							{{ column.label }}
							<Sort
								:order="sortBy[column.field]"
								:label="column.label"
								@click="sort(column.field)"
							/>
						</th>
						<th>
							<span class="is-sr-only">{{ $t('projectFinance.money') }}</span>
						</th>
					</tr>
				</thead>
				<tbody>
					<tr
						v-for="row in rows"
						:key="row.id"
						:class="{'is-archived': row.is_archived}"
					>
						<td>
							<RouterLink
								:to="{name: 'project.index', params: {projectId: row.id}}"
								class="projects-overview__project"
							>
								<span
									v-if="row.hex_color"
									class="projects-overview__dot"
									:style="{backgroundColor: `#${row.hex_color}`}"
								/>
								{{ row.title }}
							</RouterLink>
							<span
								v-if="row.is_archived"
								class="tag is-light ml-2"
							>{{ $t('projectFinance.overview.archived') }}</span>
						</td>
						<td>
							<span v-if="row.client">{{ row.client }}</span>
							<span
								v-else
								class="has-text-grey-light"
							>—</span>
						</td>
						<td>
							<span
								v-for="worker in row.workers"
								:key="worker.id"
								v-tooltip="$t('projectFinance.overview.workerOpenTasks', {count: worker.open_tasks})"
								class="tag projects-overview__worker"
								:class="{'is-primary is-light': worker.id === myId}"
							>
								{{ worker.name || worker.username }}
								<span
									v-if="worker.open_tasks > 0"
									class="projects-overview__worker-count"
								>{{ worker.open_tasks }}</span>
							</span>
							<span
								v-if="row.workers.length === 0"
								class="has-text-grey-light"
							>—</span>
						</td>
						<td class="has-text-right">
							<RouterLink
								v-if="row.my_open_tasks > 0"
								v-tooltip="$t('projectFinance.overview.myTasksTooltip')"
								:to="{name: 'project.index', params: {projectId: row.id}}"
								class="tag is-warning is-light"
							>
								{{ $t('projectFinance.overview.mine', {count: row.my_open_tasks}) }}
							</RouterLink>
							{{ $t('projectFinance.overview.tasksProgress', {open: row.open_tasks, done: row.done_tasks}) }}
						</td>
						<td class="has-text-right">
							{{ row.fee ? formatMoney(row.fee, row.currency) : '—' }}
						</td>
						<td class="has-text-right">
							{{ row.paid ? formatMoney(row.paid, row.currency) : '—' }}
						</td>
						<td
							class="has-text-right"
							:class="{'has-text-danger has-text-weight-semibold': row.owed > 0, 'has-text-success': row.fee > 0 && row.owed === 0}"
						>
							{{ row.fee ? (row.owed > 0 ? formatMoney(row.owed, row.currency) : $t('projectFinance.overview.paidInFull')) : '—' }}
						</td>
						<td class="has-text-right">
							<XButton
								variant="tertiary"
								icon="coins"
								:to="{name: 'project.settings.finance', params: {projectId: row.id}}"
							>
								{{ $t('projectFinance.money') }}
							</XButton>
						</td>
					</tr>
				</tbody>
			</table>
		</div>
	</div>
</template>

<script setup lang="ts">
import {computed, ref, watch} from 'vue'
import {useI18n} from 'vue-i18n'

import XButton from '@/components/input/Button.vue'
import FormInput from '@/components/input/FormInput.vue'
import FormSelect from '@/components/input/FormSelect.vue'
import Sort from '@/components/tasks/partials/Sort.vue'

import {formatMoney, totalsByCurrency} from '@/client/queries/projectFinance'
import type {ProjectOverviewResponse} from '@/client/queries/projectFinance'
import {useProjectOverview} from '@/composables/useProjectOverview'
import {useDelayedLoading} from '@/composables/useDelayedLoading'
import {useTableSort, type TableSortState} from '@/composables/useTableSort'
import {useTitle} from '@/composables/useTitle'
import {useAuthStore} from '@/stores/auth'

defineOptions({name: 'ProjectsOverview'})

const {t, locale} = useI18n({useScope: 'global'})
useTitle(() => t('projectFinance.overview.title'))

const authStore = useAuthStore()
const myId = computed(() => authStore.info?.id ?? 0)

const {items, isPending, isFetching} = useProjectOverview()
const showLoading = useDelayedLoading(() => isPending.value || isFetching.value)

type SortField = 'title' | 'client' | 'workers' | 'tasks' | 'fee' | 'paid' | 'owed'
type StatusFilter = 'active' | 'archived' | 'owed' | 'all'

const STORAGE_KEY = 'projectsOverviewFilters'
type StoredFilters = {client: string, worker: number, status: StatusFilter, sortBy: TableSortState<SortField>}

function loadFilters(): Partial<StoredFilters> {
	try {
		return JSON.parse(localStorage.getItem(STORAGE_KEY) ?? '{}') as Partial<StoredFilters>
	} catch {
		return {}
	}
}

const stored = loadFilters()
const search = ref('')
const clientFilter = ref<string>(stored.client ?? '')
const workerFilter = ref<number>(stored.worker ?? 0)
const statusFilter = ref<StatusFilter>(stored.status ?? 'active')
const sortBy = ref<TableSortState<SortField>>(stored.sortBy ?? {client: 'asc'})

watch([clientFilter, workerFilter, statusFilter, sortBy], () => {
	try {
		localStorage.setItem(STORAGE_KEY, JSON.stringify({
			client: clientFilter.value,
			worker: workerFilter.value,
			status: statusFilter.value,
			sortBy: sortBy.value,
		}))
	} catch {
		// Storage unavailable (private window); filters just won't be remembered.
	}
})

const {sort, ariaSort} = useTableSort<SortField>(sortBy, () => {})

const columns = computed<{field: SortField, label: string, numeric?: boolean}[]>(() => [
	{field: 'title', label: t('project.title')},
	{field: 'client', label: t('projectFinance.client')},
	{field: 'workers', label: t('projectFinance.overview.workers')},
	{field: 'tasks', label: t('projectFinance.overview.tasks'), numeric: true},
	{field: 'fee', label: t('projectFinance.fee'), numeric: true},
	{field: 'paid', label: t('projectFinance.paid'), numeric: true},
	{field: 'owed', label: t('projectFinance.owed'), numeric: true},
])

const activeCount = computed(() => items.value.filter(item => !item.is_archived).length)
const archivedCount = computed(() => items.value.filter(item => item.is_archived).length)

const clientOptions = computed(() => {
	const clients = [...new Set(items.value.map(item => item.client).filter(Boolean))]
		.sort((a, b) => a.localeCompare(b, locale.value))
	return [
		{value: '', label: t('projectFinance.overview.allClients')},
		...clients.map(client => ({value: client, label: client})),
		{value: '__none__', label: t('projectFinance.overview.noClient')},
	]
})

const workerOptions = computed(() => {
	const workers = new Map<number, string>()
	for (const item of items.value) {
		for (const worker of item.workers) {
			workers.set(worker.id, worker.name || worker.username)
		}
	}
	return [
		{value: 0, label: t('projectFinance.overview.allWorkers')},
		...[...workers.entries()]
			.sort((a, b) => a[1].localeCompare(b[1], locale.value))
			.map(([id, name]) => ({value: id, label: id === myId.value ? t('projectFinance.overview.me', {name}) : name})),
	]
})

const statusOptions = computed(() => [
	{value: 'active', label: t('projectFinance.overview.statusActive')},
	{value: 'owed', label: t('projectFinance.overview.statusOwed')},
	{value: 'archived', label: t('projectFinance.overview.statusArchived')},
	{value: 'all', label: t('projectFinance.overview.statusAll')},
])

const filtered = computed(() => {
	const needle = search.value.trim().toLowerCase()
	return items.value.filter(item => {
		if (statusFilter.value === 'active' && item.is_archived) return false
		if (statusFilter.value === 'archived' && !item.is_archived) return false
		if (statusFilter.value === 'owed' && item.owed <= 0) return false
		if (clientFilter.value === '__none__' && item.client) return false
		if (clientFilter.value && clientFilter.value !== '__none__' && item.client !== clientFilter.value) return false
		if (workerFilter.value && !item.workers.some(worker => worker.id === workerFilter.value)) return false
		if (needle && !`${item.title} ${item.client}`.toLowerCase().includes(needle)) return false
		return true
	})
})

const totals = computed(() => totalsByCurrency(filtered.value))

function workersKey(item: ProjectOverviewResponse): string {
	return item.workers.map(worker => worker.name || worker.username).join(', ')
}

function compare(a: ProjectOverviewResponse, b: ProjectOverviewResponse, field: SortField): number {
	switch (field) {
		case 'title':
			return a.title.localeCompare(b.title, locale.value)
		case 'client':
			// Projects without a client always go last.
			if (!a.client !== !b.client) return a.client ? -1 : 1
			return a.client.localeCompare(b.client, locale.value) || a.title.localeCompare(b.title, locale.value)
		case 'workers':
			if (!a.workers.length !== !b.workers.length) return a.workers.length ? -1 : 1
			return workersKey(a).localeCompare(workersKey(b), locale.value)
		case 'tasks':
			return a.open_tasks - b.open_tasks
		case 'fee':
			return a.fee - b.fee
		case 'paid':
			return a.paid - b.paid
		case 'owed':
			return a.owed - b.owed
	}
}

const rows = computed(() => {
	const entries = Object.entries(sortBy.value).filter(([, order]) => order === 'asc' || order === 'desc') as [SortField, 'asc' | 'desc'][]
	if (entries.length === 0) {
		return filtered.value
	}
	return [...filtered.value].sort((a, b) => {
		for (const [field, order] of entries) {
			const result = compare(a, b, field)
			if (result !== 0) return order === 'asc' ? result : -result
		}
		return 0
	})
})
</script>

<style lang="scss" scoped>
.projects-overview__stats {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(12rem, 1fr));
	gap: 1rem;
	margin-block-end: 1.5rem;
}

.projects-overview__stat {
	display: flex;
	flex-direction: column;
	gap: .25rem;
	padding: 1rem;
	background: var(--white);
	border-radius: $radius;
	box-shadow: var(--shadow-sm);
}

.projects-overview__stat-label {
	font-size: .8rem;
	color: var(--grey-500);
	text-transform: uppercase;
	letter-spacing: .03em;
}

.projects-overview__stat-value {
	font-size: 1.6rem;
	font-weight: 700;
	font-variant-numeric: tabular-nums;
}

.projects-overview__stat-sub {
	font-size: .8rem;
	color: var(--grey-500);
}

.projects-overview__filters {
	display: flex;
	flex-wrap: wrap;
	gap: .75rem;
	margin-block-end: 1rem;
}

.projects-overview__search {
	flex: 1 1 14rem;
}

.projects-overview__empty {
	padding: 2rem 0;
}

.projects-overview__table-wrap {
	overflow-x: auto;
	background: var(--white);
	border-radius: $radius;
	box-shadow: var(--shadow-sm);
}

.projects-overview__table {
	td, th {
		vertical-align: middle;
		white-space: nowrap;
	}

	td {
		font-variant-numeric: tabular-nums;
	}

	tr.is-archived td {
		opacity: .65;
	}
}

.projects-overview__project {
	display: inline-flex;
	align-items: center;
	gap: .5rem;
	font-weight: 600;
}

.projects-overview__dot {
	inline-size: .65rem;
	block-size: .65rem;
	border-radius: 50%;
	flex-shrink: 0;
}

.projects-overview__worker {
	margin-inline-end: .25rem;
}

.projects-overview__worker-count {
	margin-inline-start: .35rem;
	font-weight: 700;
}
</style>
