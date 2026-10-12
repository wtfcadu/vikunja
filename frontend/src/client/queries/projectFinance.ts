import {queryOptions, useMutation} from '@tanstack/vue-query'

import {
	projectFinanceRead,
	projectFinanceUpdate,
	projectOverviewList,
	projectPaymentsCreate,
	projectPaymentsDelete,
	projectPaymentsList,
	projectPaymentsUpdate,
} from '@/client/generated'
import type {
	ProjectFinance,
	ProjectOverviewItem,
	ProjectOverviewWorker,
	ProjectPayment,
} from '@/client/generated'
import {fetchAllPages} from './fetchAllPages'
import {contextMutationOptions} from './contextMutation'
import {i18n} from '@/i18n'

export const projectFinanceKeys = {
	all: ['projectFinance'] as const,
	overview: () => [...projectFinanceKeys.all, 'overview'] as const,
	finance: (projectId: number) => [...projectFinanceKeys.all, 'finance', projectId] as const,
	payments: (projectId: number) => [...projectFinanceKeys.all, 'payments', projectId] as const,
}

export type ProjectFinanceResponse = Required<Pick<ProjectFinance, 'project_id' | 'client' | 'currency' | 'fee'>>

export type ProjectPaymentResponse = Required<Pick<ProjectPayment, 'id' | 'project_id' | 'amount' | 'paid_at' | 'note'>>

export type ProjectOverviewWorkerResponse = Required<Pick<ProjectOverviewWorker, 'id' | 'username' | 'name' | 'open_tasks'>>

export type ProjectOverviewResponse = Required<Omit<ProjectOverviewItem, '$schema' | 'workers'>> & {
	workers: ProjectOverviewWorkerResponse[]
}

export function normalizeProjectFinance(projectId: number, finance: Partial<ProjectFinance> | undefined): ProjectFinanceResponse {
	return {
		project_id: finance?.project_id ?? projectId,
		client: finance?.client ?? '',
		currency: finance?.currency ?? '',
		fee: finance?.fee ?? 0,
	}
}

export function normalizeProjectPayment(payment: ProjectPayment): ProjectPaymentResponse {
	return {
		id: payment.id ?? 0,
		project_id: payment.project_id ?? 0,
		amount: payment.amount ?? 0,
		paid_at: payment.paid_at ?? new Date().toISOString(),
		note: payment.note ?? '',
	}
}

export function normalizeProjectOverviewItem(item: ProjectOverviewItem): ProjectOverviewResponse {
	return {
		id: item.id ?? 0,
		title: item.title ?? '',
		hex_color: item.hex_color ?? '',
		is_archived: item.is_archived ?? false,
		parent_project_id: item.parent_project_id ?? 0,
		client: item.client ?? '',
		currency: item.currency ?? '',
		fee: item.fee ?? 0,
		paid: item.paid ?? 0,
		owed: item.owed ?? 0,
		payments: item.payments ?? 0,
		open_tasks: item.open_tasks ?? 0,
		done_tasks: item.done_tasks ?? 0,
		my_open_tasks: item.my_open_tasks ?? 0,
		workers: (item.workers ?? []).map(worker => ({
			id: worker.id ?? 0,
			username: worker.username ?? '',
			name: worker.name ?? '',
			open_tasks: worker.open_tasks ?? 0,
		})),
	}
}

/** Converts minor units (pence, cents) to a display string such as "£1,250.00". */
export function formatMoney(minor: number, currency: string, locale: string = i18n.global.locale.value): string {
	const value = minor / 100
	if (/^[A-Z]{3}$/.test(currency)) {
		try {
			return new Intl.NumberFormat(locale, {style: 'currency', currency}).format(value)
		} catch {
			// Unknown currency code: fall through to a plain number.
		}
	}
	const number = new Intl.NumberFormat(locale, {minimumFractionDigits: 2, maximumFractionDigits: 2}).format(value)
	return currency ? `${number} ${currency}` : number
}

/** Parses what someone typed ("1250", "1,250.50", "1250,5") into minor units. */
export function parseMoney(input: string | number): number {
	if (typeof input === 'number') {
		return Math.max(0, Math.round(input * 100))
	}
	let cleaned = input.trim().replace(/[^\d.,-]/g, '')
	const lastComma = cleaned.lastIndexOf(',')
	const lastDot = cleaned.lastIndexOf('.')
	if (lastComma > lastDot) {
		// Comma is the decimal separator ("1.250,50" or "1250,5").
		cleaned = cleaned.replace(/\./g, '').replace(',', '.')
	} else {
		cleaned = cleaned.replace(/,/g, '')
	}
	const value = Number.parseFloat(cleaned)
	if (!Number.isFinite(value)) {
		return 0
	}
	return Math.max(0, Math.round(value * 100))
}

/** Minor units back to an editable string without currency symbols, e.g. "1250.00". */
export function minorToInput(minor: number): string {
	return (minor / 100).toFixed(2)
}

export function projectOverviewQuery() {
	return queryOptions({
		queryKey: projectFinanceKeys.overview(),
		queryFn: async () => {
			const {data} = await projectOverviewList()
			return (data?.items ?? []).map(normalizeProjectOverviewItem)
		},
		staleTime: 30 * 1000,
	})
}

export function projectFinanceQuery(projectId: number) {
	return queryOptions({
		queryKey: projectFinanceKeys.finance(projectId),
		queryFn: async () => {
			const {data} = await projectFinanceRead({path: {project: projectId}})
			return normalizeProjectFinance(projectId, data)
		},
		meta: {handlesError: true},
	})
}

export function projectPaymentsQuery(projectId: number) {
	return queryOptions({
		queryKey: projectFinanceKeys.payments(projectId),
		queryFn: async () => {
			const payments = await fetchAllPages(async page =>
				(await projectPaymentsList({path: {project: projectId}, query: {page}})).data,
			)
			return payments.map(normalizeProjectPayment)
		},
		meta: {handlesError: true},
	})
}

export type SaveProjectFinanceInput = ProjectFinanceResponse

export function saveProjectFinanceMutationOptions() {
	return contextMutationOptions({
		mutationFn: async (input: SaveProjectFinanceInput) => {
			const {data} = await projectFinanceUpdate({
				path: {project: input.project_id},
				body: {client: input.client, currency: input.currency.toUpperCase(), fee: input.fee},
			})
			return normalizeProjectFinance(input.project_id, data)
		},
		onSuccess: (saved, input, client) => {
			client.setQueryData(projectFinanceKeys.finance(input.project_id), saved)
		},
		onSettled: (input, client) => Promise.all([
			client.invalidateQueries({queryKey: projectFinanceKeys.finance(input.project_id), exact: true}),
			client.invalidateQueries({queryKey: projectFinanceKeys.overview(), exact: true}),
		]),
		successMessage: () => i18n.global.t('projectFinance.saved'),
	})
}

export type SaveProjectPaymentInput = {
	projectId: number
	id?: number
	amount: number
	paidAt: string
	note: string
}

export function saveProjectPaymentMutationOptions() {
	return contextMutationOptions({
		mutationFn: async (input: SaveProjectPaymentInput) => {
			const body = {amount: input.amount, paid_at: input.paidAt, note: input.note}
			if (input.id) {
				const {data} = await projectPaymentsUpdate({path: {project: input.projectId, payment: input.id}, body})
				return normalizeProjectPayment(data)
			}
			const {data} = await projectPaymentsCreate({path: {project: input.projectId}, body})
			return normalizeProjectPayment(data)
		},
		onSettled: (input, client) => Promise.all([
			client.invalidateQueries({queryKey: projectFinanceKeys.payments(input.projectId), exact: true}),
			client.invalidateQueries({queryKey: projectFinanceKeys.overview(), exact: true}),
		]),
		successMessage: (_data, input) => input.id
			? i18n.global.t('projectFinance.payment.updated')
			: i18n.global.t('projectFinance.payment.added'),
	})
}

export type DeleteProjectPaymentInput = {projectId: number, id: number}

export function deleteProjectPaymentMutationOptions() {
	return contextMutationOptions({
		mutationFn: async (input: DeleteProjectPaymentInput) => {
			await projectPaymentsDelete({path: {project: input.projectId, payment: input.id}})
			return input.id
		},
		onSuccess: (_id, input, client) => {
			client.setQueryData<ProjectPaymentResponse[]>(
				projectFinanceKeys.payments(input.projectId),
				current => current?.filter(payment => payment.id !== input.id),
			)
		},
		onSettled: (input, client) => Promise.all([
			client.invalidateQueries({queryKey: projectFinanceKeys.payments(input.projectId), exact: true}),
			client.invalidateQueries({queryKey: projectFinanceKeys.overview(), exact: true}),
		]),
		successMessage: () => i18n.global.t('projectFinance.payment.deleted'),
	})
}

export function useSaveProjectFinanceMutation() {
	return useMutation(saveProjectFinanceMutationOptions())
}

export function useSaveProjectPaymentMutation() {
	return useMutation(saveProjectPaymentMutationOptions())
}

export function useDeleteProjectPaymentMutation() {
	return useMutation(deleteProjectPaymentMutationOptions())
}

/** Totals per currency over a set of overview rows. */
export function totalsByCurrency(items: ProjectOverviewResponse[]) {
	const totals = new Map<string, {currency: string, fee: number, paid: number, owed: number, projects: number}>()
	for (const item of items) {
		const currency = item.currency || ''
		const entry = totals.get(currency) ?? {currency, fee: 0, paid: 0, owed: 0, projects: 0}
		entry.fee += item.fee
		entry.paid += item.paid
		entry.owed += item.owed
		entry.projects++
		totals.set(currency, entry)
	}
	return [...totals.values()]
		.filter(total => total.fee > 0 || total.paid > 0)
		.sort((a, b) => a.currency.localeCompare(b.currency))
}
