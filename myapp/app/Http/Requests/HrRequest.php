<?php

namespace App\Http\Requests;

use Illuminate\Foundation\Http\FormRequest;

class HrRequest extends FormRequest
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
            'hr_cd' => 'unique:mst_hr|max:16',
            'user_name' => 'required|string|max:128',
            'name_kana' => 'string|max:128',
            'is_admin' => 'required|integer',
            'remarks' => 'max:256',
            'prices.*.role_id' => 'required',
            'prices.*.price' => 'required|integer|max:2000000000|min:0',
            'prices.*.from_date' => 'required|date',
        ];
    }
}
