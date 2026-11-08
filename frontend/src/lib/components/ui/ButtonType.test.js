import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import Button from './Button.svelte';

describe('Button type override', () => {
	it('has default type button', () => {
		const { getByRole } = render(Button);
		expect(getByRole('button').getAttribute('type')).toBe('button');
	});

	it('can be overridden to submit', () => {
		const { getByRole } = render(Button, { props: { type: 'submit' } });
		expect(getByRole('button').getAttribute('type')).toBe('submit');
	});
});
