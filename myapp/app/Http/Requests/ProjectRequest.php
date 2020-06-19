<?php

namespace App\Http\Requests;

use Illuminate\Foundation\Http\FormRequest;

class ProjectRequest extends FormRequest
{
    /**
     * Determine if the user is authorized to make this request.
     *
     * @return bool
     */
    public function authorize()
    {
        return true;
    }

    /**
     * Get the validation rules that apply to the request.
     *
     * @return array
     */
    public function rules()
    {
        return [
            'project_no' => 'required|unique:trn_project|max:32',
            'project_name' => 'required|max:64',
            'order_amount' => 'numeric',
            'from_date' => 'nullable|date',
            'to_date' => 'nullable|date',
        ];
    }

    public function prepareForValidation()
    {
        $this->merge([
            'order_amount' => preg_replace('/[￥ ,]/', '', $this->order_amount)
        ]);
    }
}
